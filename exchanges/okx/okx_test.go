package okx

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = true
)

var ok Okx
var wsSetupRan bool
var wsSetupLocker sync.Mutex

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	exchCfg, err := cfg.GetExchangeConfig("Okx")
	if err != nil {
		log.Fatal(err)
	}
	ok.SkipAuthCheck = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.ClientID = passphrase
	ok.WsResponseMultiplexer = wsRequestDataChannelsMultiplexer{
		WsResponseChannelsMap: make(map[string]*wsRequestInfo),
		Register:              make(chan *wsRequestInfo),
		Unregister:            make(chan string),
		Message:               make(chan *wsIncomingData),
	}
	ok.SetDefaults()

	if apiKey != "" && apiSecret != "" {
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
	}
	err = ok.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	ok.Websocket = sharedtestvalues.NewTestWebsocket()
	ok.Base.Config = exchCfg
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

var marketDataResponseJSON = `{"instType": "SWAP","instId": "LTC-USD-SWAP","last": "9999.99","lastSz": "0.1","askPx": "9999.99","askSz": "11","bidPx": "8888.88","bidSz": "5","open24h": "9000","high24h": "10000","low24h": "8888.88","volCcy24h": "2222","vol24h": "2222","sodUtc0": "2222","sodUtc8": "2222","ts": "1597026383085"}`

func TestGetTickers(t *testing.T) {
	t.Parallel()
	var resp TickerResponse
	if err := json.Unmarshal([]byte(marketDataResponseJSON), &resp); err != nil {
		t.Error("Okx decerializing to MarketDataResponse error", err)
	}
	_, err := ok.GetTickers(context.Background(), "OPTION", "", "SOL-USD")
	if err != nil {
		t.Error("Okx GetTickers() error", err)
	}
}

func TestGetIndexTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.GetIndexTickers(context.Background(), "USDT", "")
	if err != nil {
		t.Error("OKX GetIndexTicker() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetTicker(context.Background(), "NEAR-USDT-SWAP"); err != nil {
		t.Error("Okx GetTicker() error", err)
	}
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOrderBookDepth(context.Background(), "BTC-USDT", 10)
	if err != nil {
		t.Error("OKX GetOrderBookDepth() error", err)
	}
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCandlesticks(context.Background(), "BTC-USDT", kline.OneHour, time.Now().Add(-time.Hour*2), time.Now(), 30)
	if err != nil {
		t.Error("Okx GetCandlesticks() error", err)
	}
}
func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	futuresPairs, err := ok.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Errorf("%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	}
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	currencyPair, err := currency.NewPairFromString(futuresPairs[0])
	if err != nil {
		t.Error(err)
	}
	if _, err := ok.GetHistoricCandlesExtended(context.Background(), currencyPair, asset.Futures, time.Now().Add(-time.Hour*24), time.Now(), kline.OneMin); err != nil {
		t.Errorf("%s GetHistoricCandlesExtended() error: %v", ok.Name, err)
	}
}

func TestGetCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCandlesticksHistory(context.Background(), "BTC-USDT", kline.OneHour, time.Unix(time.Now().Unix()-3600, 0), time.Now(), 30)
	if err != nil {
		t.Error("Okx GetCandlesticksHistory() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTrades(context.Background(), "BTC-USDT", 30)
	if err != nil {
		t.Error("Okx GetTrades() error", err)
	}
}

var tradeHistoryJSON = `{"instId": "BTC-USDT","side": "sell","sz": "0.00001","px": "29963.2","tradeId": "242720720","ts": "1654161646974"}`

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	var resp TradeResponse
	if err := json.Unmarshal([]byte(tradeHistoryJSON), &resp); err != nil {
		t.Error("Okx decerializing to TradeResponse struct error", err)
	}
	if _, err := ok.GetTradesHistory(context.Background(), "BTC-USDT", "", "", 0); err != nil {
		t.Error("Okx GetTradeHistory() error", err)
	}
}

func TestGet24HTotalVolume(t *testing.T) {
	t.Parallel()
	_, err := ok.Get24HTotalVolume(context.Background())
	if err != nil {
		t.Error("Okx Get24HTotalVolume() error", err)
	}
}

func TestGetOracle(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOracle(context.Background())
	if err != nil {
		t.Error("Okx GetOracle() error", err)
	}
}

func TestGetExchangeRate(t *testing.T) {
	t.Parallel()
	_, err := ok.GetExchangeRate(context.Background())
	if err != nil {
		t.Error("Okx GetExchangeRate() error", err)
	}
}

func TestGetIndexComponents(t *testing.T) {
	t.Parallel()
	_, err := ok.GetIndexComponents(context.Background(), "ETH-USDT")
	if err != nil {
		t.Error("Okx GetIndexComponents() error", err)
	}
}

var blockTickerItemJSON = `{"instType":"SWAP","instId":"LTC-USD-SWAP","volCcy24h":"2222","vol24h":"2222","ts":"1597026383085"}`

func TestGetBlockTickers(t *testing.T) {
	t.Parallel()
	var resp BlockTicker
	if err := json.Unmarshal([]byte(blockTickerItemJSON), &resp); err != nil {
		t.Error("Okx Decerializing to BlockTickerItem error", err)
	}
	if _, err := ok.GetBlockTickers(context.Background(), "SWAP", ""); err != nil {
		t.Error("Okx GetBlockTickers() error", err)
	}
}

func TestGetBlockTicker(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetBlockTicker(context.Background(), "BTC-USDT"); err != nil {
		t.Error("Okx GetBlockTicker() error", err)
	}
}

var blockTradeItemJSON = `{"instId":"BTC-USDT-SWAP","tradeId":"90167","px":"42000","sz":"100","side":"sell","ts":"1642670926504"}`

func TestGetBlockTrade(t *testing.T) {
	t.Parallel()
	var resp BlockTrade
	if err := json.Unmarshal([]byte(blockTradeItemJSON), &resp); err != nil {
		t.Error("Okx Decerializing to BlockTrade error", err)
	}
	if _, err := ok.GetBlockTrades(context.Background(), "BTC-USDT"); err != nil {
		t.Error("Okx GetBlockTrades() error", err)
	}
}

func TestGetInstrument(t *testing.T) {
	t.Parallel()
	_, err := ok.GetInstruments(context.Background(), &InstrumentsFetchParams{
		InstrumentType: "OPTION",
		Underlying:     "SOL-USD",
	})
	if err != nil {
		t.Error("Okx GetInstruments() error", err)
	}
	_, err = ok.GetInstruments(context.Background(), &InstrumentsFetchParams{
		InstrumentType: "OPTION",
		Underlying:     "SOL-USD",
	})
	if err != nil {
		t.Error("Okx GetInstruments() error", err)
	}
}

var deliveryHistoryData = `[{"ts":"1597026383085","details":[{"type":"delivery","insId":"ZIL-BTC","px":"0.016"}]},{"ts":"1597026383085","details":[{"insId":"BTC-USD-200529-6000-C","type":"exercised","px":"0.016"},{"insId":"BTC-USD-200529-8000-C","type":"exercised","px":"0.016"}]}]`

func TestGetDeliveryHistory(t *testing.T) {
	t.Parallel()
	var repo []DeliveryHistory
	if err := json.Unmarshal([]byte(deliveryHistoryData), &repo); err != nil {
		t.Error("Okx error", err)
	}
	_, err := ok.GetDeliveryHistory(context.Background(), "FUTURES", "BTC-USDT", time.Time{}, time.Time{}, 100)
	if err != nil {
		t.Error("okx GetDeliveryHistory() error", err)
	}
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetOpenInterest(context.Background(), "FUTURES", "BTC-USDT", ""); err != nil {
		t.Error("Okx GetOpenInterest() error", err)
	}
}

func TestGetFundingRate(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetFundingRate(context.Background(), "BTC-USD-SWAP"); err != nil {
		t.Error("okx GetFundingRate() error", err)
	}
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetFundingRateHistory(context.Background(), "BTC-USD-SWAP", time.Time{}, time.Time{}, 10); err != nil {
		t.Error("Okx GetFundingRateHistory() error", err)
	}
}

func TestGetLimitPrice(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetLimitPrice(context.Background(), "BTC-USD-SWAP"); err != nil {
		t.Error("okx GetLimitPrice() error", err)
	}
}

func TestGetOptionMarketData(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetOptionMarketData(context.Background(), "BTC-USD", time.Time{}); err != nil {
		t.Error("Okx GetOptionMarketData() error", err)
	}
}

var estimatedDeliveryResponseString = `[{"instType":"FUTURES","instId":"BTC-USDT-201227","settlePx":"200","ts":"1597026383085"}]`

func TestGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	var result []DeliveryEstimatedPrice
	err := json.Unmarshal([]byte(estimatedDeliveryResponseString), (&result))
	if err != nil {
		t.Error("Okx GetEstimatedDeliveryPrice() error", err)
	}
	if _, err := ok.GetEstimatedDeliveryPrice(context.Background(), "BTC-USD"); err != nil && !(strings.Contains(err.Error(), "Instrument ID does not exist.")) {
		t.Error("Okx GetEstimatedDeliveryPrice() error", err)
	}
}

var discountRateJSON = `{"amt":"1","ccy":"BTC","discountInfo":[{"discountRate":"1","maxAmt":"5000000","minAmt":"0"},{"discountRate":"0.975","maxAmt":"10000000","minAmt":"5000000"},{"discountRate":"0.975","maxAmt":"20000000","minAmt":"10000000"},{"discountRate":"0.95","maxAmt":"40000000","minAmt":"20000000"},{"discountRate":"0.9","maxAmt":"100000000","minAmt":"40000000"},{"discountRate":"0","maxAmt":"","minAmt":"100000000"}],"discountLv":"1"}`

func TestGetDiscountRateAndInterestFreeQuota(t *testing.T) {
	t.Parallel()
	var resp DiscountRate
	if err := json.Unmarshal([]byte(discountRateJSON), &resp); err != nil {
		t.Errorf("%s error while deserializing to DiscountRate %v", ok.Name, err)
	}
	_, err := ok.GetDiscountRateAndInterestFreeQuota(context.Background(), "", 0)
	if err != nil {
		t.Error("Okx GetDiscountRateAndInterestFreeQuota() error", err)
	}
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetSystemTime(context.Background()); err != nil {
		t.Error("Okx GetSystemTime() error", err)
	}
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetLiquidationOrders(context.Background(), &LiquidationOrderRequestParams{
		InstrumentType: "MARGIN",
		Underlying:     "BTC-USD",
		Currency:       currency.BTC,
	}); err != nil {
		t.Error("Okx GetLiquidationOrders() error", err)
	}
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetMarkPrice(context.Background(), "MARGIN", "", ""); err != nil {
		t.Error("Okx GetMarkPrice() error", err)
	}
}

func TestGetPositionTiers(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetPositionTiers(context.Background(), "FUTURES", "cross", "BTC-USDT", "", ""); err != nil {
		t.Error("Okx GetPositionTiers() error", err)
	}
}

func TestGetInterestRateAndLoanQuota(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetInterestRateAndLoanQuota(context.Background()); err != nil {
		t.Error("Okx GetInterestRateAndLoanQuota() error", err)
	}
}

func TestGetInterestRateAndLoanQuotaForVIPLoans(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetInterestRateAndLoanQuotaForVIPLoans(context.Background()); err != nil {
		t.Error("Okx GetInterestRateAndLoanQuotaForVIPLoans() error", err)
	}
}

func TestGetPublicUnderlyings(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetPublicUnderlyings(context.Background(), "swap"); err != nil {
		t.Error("Okx GetPublicUnderlyings() error", err)
	}
}

func TestGetInsuranceFundInformations(t *testing.T) {
	t.Parallel()
	// getting the Underlyings usig the Get public Underlyinggs method for specific instrument type.
	var underlyings []string
	var err error
	if underlyings, err = ok.GetPublicUnderlyings(context.Background(), "futures"); err != nil {
		t.Error("Okx GetPublicUnderlyings() error", err)
		t.SkipNow()
	}
	if _, err := ok.GetInsuranceFundInformations(context.Background(), &InsuranceFundInformationRequestParams{
		InstrumentType: "FUTURES",
		Underlying:     underlyings[0],
	}); err != nil {
		t.Error("Okx GetInsuranceFundInformations() error", err)
	}
}

var currencyConvertJSON = `{
	"instId": "BTC-USD-SWAP",
	"px": "35000",
	"sz": "311",
	"type": "1",
	"unit": "coin"
}`

func TestCurrencyUnitConvert(t *testing.T) {
	t.Parallel()
	var resp UnitConvertResponse
	if err := json.Unmarshal([]byte(currencyConvertJSON), &resp); err != nil {
		t.Error("Okx Decerializing to UnitConvertResponse error", err)
	}
	if _, err := ok.CurrencyUnitConvert(context.Background(), "BTC-USD-SWAP", 1, 3500, CurrencyToContract, ""); err != nil {
		t.Error("Okx CurrencyUnitConvert() error", err)
	}
}

// Trading related enndpoints test functions.
func TestGetSupportCoins(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetSupportCoins(context.Background()); err != nil {
		t.Error("Okx GetSupportCoins() error", err)
	}
}

func TestGetTakerVolume(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetTakerVolume(context.Background(), "BTC", "SPOT", time.Time{}, time.Time{}, kline.OneDay); err != nil {
		t.Error("Okx GetTakerVolume() error", err)
	}
}
func TestGetMarginLendingRatio(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetMarginLendingRatio(context.Background(), "BTC", time.Time{}, time.Time{}, kline.OneDay); err != nil {
		t.Error("Okx GetMarginLendingRatio() error", err)
	}
}

func TestGetLongShortRatio(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetLongShortRatio(context.Background(), "BTC", time.Time{}, time.Time{}, kline.OneDay); err != nil {
		t.Error("Okx GetLongShortRatio() error", err)
	}
}

func TestGetContractsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetContractsOpenInterestAndVolume(context.Background(), "BTC", time.Time{}, time.Time{}, kline.OneDay); err != nil {
		t.Error("Okx GetContractsOpenInterestAndVolume() error", err)
	}
}

func TestGetOptionsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetOptionsOpenInterestAndVolume(context.Background(), "BTC", kline.OneDay); err != nil {
		t.Error("Okx GetOptionsOpenInterestAndVolume() error", err)
	}
}

func TestGetPutCallRatio(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetPutCallRatio(context.Background(), "BTC", kline.OneDay); err != nil {
		t.Error("Okx GetPutCallRatio() error", err)
	}
}

func TestGetOpenInterestAndVolumeExpiry(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetOpenInterestAndVolumeExpiry(context.Background(), "BTC", kline.OneDay); err != nil {
		t.Error("Okx GetOpenInterestAndVolume() error", err)
	}
}

func TestGetOpenInterestAndVolumeStrike(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetOpenInterestAndVolumeStrike(context.Background(), "BTC", time.Now(), kline.OneDay); err != nil {
		t.Error("Okx GetOpenInterestAndVolumeStrike() error", err)
	}
}

func TestGetTakerFlow(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetTakerFlow(context.Background(), "BTC", kline.OneDay); err != nil {
		t.Error("Okx GetTakerFlow() error", err)
	}
}

var placeOrderRequestParamsJSON = `{"instId":"BTC-USDT",    "tdMode":"cash",    "clOrdId":"b15",    "side":"buy",    "ordType":"limit",    "px":"2.15",    "sz":"2"}`

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	var resp PlaceOrderRequestParam
	if err := json.Unmarshal([]byte(placeOrderRequestParamsJSON), &resp); err != nil {
		t.Errorf("%s error while deserializing to PlaceOrderRequestParam: %v", ok.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.PlaceOrder(context.Background(), &PlaceOrderRequestParam{
		ClientSupplierOrderID: "my-new-id",
		InstrumentID:          "BTC-USDC",
		TradeMode:             "cross",
		Side:                  "buy",
		OrderType:             "limit",
		QuantityToBuyOrSell:   2.6,
		Price:                 2.1,
	}, /*&resp*/ asset.Margin); err != nil && !strings.Contains(err.Error(), "Operation failed.") {
		t.Error("Okx PlaceOrder() error", err)
	}
}

func TestPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.PlaceMultipleOrders(context.Background(),
		[]PlaceOrderRequestParam{
			{
				InstrumentID:        "GNX-BTC",
				TradeMode:           "cross",
				Side:                "sell",
				OrderType:           "limit",
				QuantityToBuyOrSell: 1,
				Price:               1,
			},
		}); err != nil && !strings.Contains(err.Error(), "operation failed") {
		t.Error("Okx PlaceOrderRequestParam() error", err)
	}
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.CancelSingleOrder(context.Background(),
		CancelOrderRequestParam{
			InstrumentID: "BTC-USD-190927",
			OrderID:      "2510789768709120",
		}); err != nil && !strings.Contains(err.Error(), "Operation failed") {
		t.Error("Okx CancelOrder() error", err)
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.CancelMultipleOrders(context.Background(), []CancelOrderRequestParam{{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
	}}); err != nil && !strings.Contains(err.Error(), "Operation failed") {
		t.Error("Okx CancelMultipleOrders() error", err)
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.AmendOrder(context.Background(), &AmendOrderRequestParams{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}); err != nil && !strings.Contains(err.Error(), "Operation failed") {
		t.Error("Okx AmendOrder() error", err)
	}
}
func TestAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.AmendMultipleOrders(context.Background(), []AmendOrderRequestParams{{
		InstrumentID: "BTC-USDT",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}}); err != nil && !strings.Contains(err.Error(), "operation failed") {
		t.Error("Okx AmendMultipleOrders() error", err)
	}
}

func TestClosePositions(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.ClosePositions(context.Background(), &ClosePositionsRequestParams{
		InstrumentID: "BTC-USDT",
		MarginMode:   "cross",
	}); err != nil && !strings.Contains(err.Error(), "Operation is not supported under the current account mode") {
		t.Error("Okc ClosePositions() error", err)
	}
}

var orderDetail = `{"instType": "FUTURES","instId": "BTC-USD-200329","ccy": "","ordId": "312269865356374016","clOrdId": "b1","tag": "","px": "999","sz": "3","pnl": "5","ordType": "limit","side": "buy","posSide": "long","tdMode": "isolated","accFillSz": "0","fillPx": "0","tradeId": "0","fillSz": "0","fillTime": "0","state": "live","avgPx": "0","lever": "20","tpTriggerPx": "","tpTriggerPxType": "last","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "last","slOrdPx": "","feeCcy": "","fee": "","rebateCcy": "","rebate": "","tgtCcy":"","category": "","uTime": "1597026383085","cTime": "1597026383085"}`

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	var odetail OrderDetail
	if err := json.Unmarshal([]byte(orderDetail), &odetail); err != nil {
		t.Error("Okx OrderDetail error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetOrderDetail(context.Background(), &OrderDetailRequestParam{
		InstrumentID: "BTC-USDT",
		OrderID:      "2510789768709120",
	}); !strings.Contains(err.Error(), "Order does not exist") {
		t.Error("Okx GetOrderDetail() error", err)
	}
}

const pendingOrderItemJSON = `{"accFillSz": "0","avgPx": "","cTime": "1618235248028","category": "normal","ccy": "","clOrdId": "","fee": "0","feeCcy": "BTC","fillPx": "","fillSz": "0","fillTime": "","instId": "BTC-USDT","instType": "SPOT","lever": "5.6","ordId": "301835739059335168","ordType": "limit","pnl": "0","posSide": "net","px": "59200","rebate": "0","rebateCcy": "USDT","side": "buy","slOrdPx": "","slTriggerPx": "","slTriggerPxType": "last","state": "live","sz": "1","tag": "","tgtCcy": "","tdMode": "cross","source":"","tpOrdPx": "","tpTriggerPx": "","tpTriggerPxType": "last","tradeId": "","uTime": "1618235248028"}`

func TestGetOrderList(t *testing.T) {
	t.Parallel()
	var pending PendingOrderItem
	if err := json.Unmarshal([]byte(pendingOrderItemJSON), &pending); err != nil {
		t.Error("Okx PendingPrderItem error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetOrderList(context.Background(), &OrderListRequestParams{}); err != nil {
		t.Error("Okx GetOrderList() error", err)
	}
}

var orderHistory = `{"instType": "FUTURES","instId": "BTC-USD-200329","ccy": "","ordId": "312269865356374016","clOrdId": "b1","tag": "","px": "999","sz": "3","ordType": "limit","side": "buy","posSide": "long","tdMode": "isolated","accFillSz": "0","fillPx": "0","tradeId": "0","fillSz": "0","fillTime": "0","state": "filled","avgPx": "0","lever": "20","tpTriggerPx": "","tpTriggerPxType": "last","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "last","slOrdPx": "","feeCcy": "","fee": "","rebateCcy": "","source":"","rebate": "","tgtCcy":"","pnl": "","category": "","uTime": "1597026383085","cTime": "1597026383085"}`

func TestGet7And3MonthDayOrderHistory(t *testing.T) {
	t.Parallel()
	var history PendingOrderItem
	if err := json.Unmarshal([]byte(orderHistory), &history); err != nil {
		t.Error("Okx OrderHistory error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.Get7DayOrderHistory(context.Background(), &OrderHistoryRequestParams{
		OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"},
	}); err != nil {
		t.Error("Okx Get7DayOrderHistory() error", err)
	}
	if _, err := ok.Get3MonthOrderHistory(context.Background(), &OrderHistoryRequestParams{
		OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"},
	}); err != nil {
		t.Error("Okx Get3MonthOrderHistory() error", err)
	}
}

var transactionhistoryJSON = `{"instType":"FUTURES","instId":"BTC-USD-200329","tradeId":"123","ordId":"123445","clOrdId": "b16","billId":"1111","tag":"","fillPx":"999","fillSz":"3","side":"buy","posSide":"long","execType":"M","feeCcy":"","fee":"","ts":"1597026383085"}`

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	var transactionhist TransactionDetail
	if err := json.Unmarshal([]byte(transactionhistoryJSON), &transactionhist); err != nil {
		t.Error("Okx Transaction Detail error", err.Error())
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetTransactionDetailsLast3Days(context.Background(), &TransactionDetailRequestParams{
		InstrumentType: "MARGIN",
	}); err != nil {
		t.Error("Okx GetTransactionDetailsLast3Days() error", err)
	}
	if _, err := ok.GetTransactionDetailsLast3Months(context.Background(), &TransactionDetailRequestParams{
		InstrumentType: "MARGIN",
	}); err != nil {
		t.Error("Okx GetTransactionDetailsLast3Days() error", err)
	}
}

func TestStopOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, err := ok.PlaceStopOrder(context.Background(), &AlgoOrderParams{
		TakeProfitTriggerPriceType: "index",
		InstrumentID:               "BTC-USDT",
		OrderType:                  "move_order_stop",
		Side:                       order.Buy,
	}); err != nil && errors.Is(err, errMissingTakeProfitTriggerPrice) {
		t.Errorf("Okx StopOrderParams() expecting %v, but found %v", errMissingTakeProfitTriggerPrice, err)
	}
	if _, err := ok.PlaceStopOrder(context.Background(), &AlgoOrderParams{
		TakeProfitTriggerPriceType: "index",
		InstrumentID:               "BTC-USDT",
		OrderType:                  "move_order_stop",
		Side:                       order.Buy,
	}); err != nil && errors.Is(err, errMissingTakeProfitTriggerPrice) {
		t.Errorf("Okx StopOrderParams() expecting %v, but found %v", errMissingTakeProfitTriggerPrice, err)
	}
	if _, err := ok.PlaceStopOrder(context.Background(), &AlgoOrderParams{
		TakeProfitTriggerPriceType: "index",
		InstrumentID:               "BTC-USDT",
		OrderType:                  "conditional",
		Side:                       order.Sell,
		TradeMode:                  "isolated",
		Size:                       12,

		TakeProfitTriggerPrice: 12335,
		TakeProfitOrderPrice:   1234,
	}); err != nil && !strings.Contains(err.Error(), "Unsupported operation") {
		t.Errorf("Okx StopOrderParams() error %v", err)
	}
	if _, err := ok.PlaceTrailingStopOrder(context.Background(), &AlgoOrderParams{
		CallbackRatio: 0.01,
		InstrumentID:  "BTC-USDT",
		OrderType:     "move_order_stop",
		Side:          order.Buy,
		TradeMode:     "isolated",
		Size:          2,
		ActivePrice:   1234,
	}); err != nil && !strings.Contains(err.Error(), "Unsupported operation") {
		t.Error("Okx PlaceTrailingStopOrder error", err)
	}
	if _, err := ok.PlaceIcebergOrder(context.Background(), &AlgoOrderParams{
		PriceLimit:  100.22,
		SizeLimit:   9999.9,
		PriceSpread: "0.04",

		InstrumentID: "BTC-USDT",
		OrderType:    "iceberg",
		Side:         order.Buy,

		TradeMode: "isolated",
		Size:      6,
	}); err != nil && strings.EqualFold(err.Error(), "Unsupported operation") {
		t.Error("Okx PlaceIceburgOrder() error", err)
	}
	if _, err := ok.PlaceTWAPOrder(context.Background(), &AlgoOrderParams{
		PriceLimit:   100.22,
		SizeLimit:    9999.9,
		OrderType:    "twap",
		PriceSpread:  "0.4",
		TimeInterval: kline.ThreeDay,
	}); err != nil && !errors.Is(errMissingInstrumentID, err) {
		t.Error("Okx PlaceTWAPOrder() error", err)
	}
	if _, err := ok.PlaceTWAPOrder(context.Background(), &AlgoOrderParams{
		InstrumentID: "BTC-USDT",
		PriceLimit:   100.22,
		SizeLimit:    9999.9,
		OrderType:    "twap",
		PriceSpread:  "0.4",
		TradeMode:    "cross",
		Side:         order.Sell,
		Size:         6,
		TimeInterval: kline.ThreeDay,
	}); err != nil && !strings.Contains(err.Error(), "Unsupported operation") {
		t.Error("Okx PlaceTWAPOrder() error", err)
	}
	if _, err := ok.TriggerAlgoOrder(context.Background(), &AlgoOrderParams{
		TriggerPriceType: "mark",
		TriggerPrice:     1234,

		InstrumentID: "BTC-USDT",
		OrderType:    "trigger",
		Side:         order.Buy,
		TradeMode:    "cross",
		Size:         5,
	}); err != nil && !strings.Contains(err.Error(), "Unsupported operation") {
		t.Error("Okx TriggerAlogOrder() error", err)
	}
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.CancelAlgoOrder(context.Background(), []AlgoOrderCancelParams{
		{
			InstrumentID: "BTC-USDT",
			AlgoOrderID:  "90994943",
		},
	}); err != nil && strings.Contains(err.Error(), "Unsupported operation") {
		t.Error("Okx CancelAlgoOrder() error", err)
	}
}

func TestCancelAdvanceAlgoOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.CancelAdvanceAlgoOrder(context.Background(), []AlgoOrderCancelParams{{
		InstrumentID: "BTC-USDT",
		AlgoOrderID:  "90994943",
	}}); err != nil && !strings.Contains(err.Error(), "Operation failed.") {
		t.Error("Okx CancelAdvanceAlgoOrder() error", err)
	}
}

var algoOrderResponse = `{"instType": "FUTURES","instId": "BTC-USD-200329","ordId": "312269865356374016","ccy": "BTC","algoId": "1234","sz": "999","ordType": "oco","side": "buy","posSide": "long","tdMode": "cross","tgtCcy": "","state": "1","lever": "20","tpTriggerPx": "","tpTriggerPxType": "","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "","triggerPx": "99","triggerPxType": "last","ordPx": "12","actualSz": "","actualPx": "","actualSide": "","pxVar":"","pxSpread":"","pxLimit":"","szLimit":"","timeInterval":"","triggerTime": "1597026383085","callbackRatio":"","callbackSpread":"","activePx":"","moveTriggerPx":"","cTime": "1597026383000"}`

func TestGetAlgoOrderList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	var order AlgoOrderResponse
	if err := json.Unmarshal([]byte(algoOrderResponse), &order); err != nil {
		t.Error("Okx Unmarshaling AlgoOrder Response error", err)
	}
	if _, err := ok.GetAlgoOrderList(context.Background(), "conditional", "", "", "", "", time.Time{}, time.Time{}, 20); err != nil {
		t.Error("Okx GetAlgoOrderList() error", err)
	}
}

func TestGetAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	var order AlgoOrderResponse
	if err := json.Unmarshal([]byte(algoOrderResponse), &order); err != nil {
		t.Error("Okx Unmarshaling AlgoOrder Response error", err)
	}
	if _, err := ok.GetAlgoOrderHistory(context.Background(), "conditional", "effective", "", "", "", time.Time{}, time.Time{}, 20); err != nil {
		t.Error("Okx GetAlgoOrderList() error", err)
	}
}

func TestGetEasyConvertCurrencyList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetEasyConvertCurrencyList(context.Background()); err != nil {
		t.Errorf("%s GetEasyConvertCurrencyList() error %v", ok.Name, err)
	}
}

var placeEasyConvertJSON = `[{"fillFromSz": "6.5807127",	"fillToSz": "0.17171580105126",	"fromCcy": "ADA","status": "running","toCcy": "OKB","uTime": "1661419684687"},{"fillFromSz": "2.997",		"fillToSz": "0.1683755161661844",		"fromCcy": "USDC",		"status": "running",		"toCcy": "OKB",		"uTime": "1661419684687"	}]`

func TestPlaceEasyConvert(t *testing.T) {
	t.Parallel()
	var response []EasyConvertItem
	if err := json.Unmarshal([]byte(placeEasyConvertJSON), &response); err != nil {
		t.Errorf("%s EasyConvertItem() error %v", ok.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.PlaceEasyConvert(context.Background(),
		PlaceEasyConvertParam{
			FromCurrency: []string{"BTC"},
			ToCurrency:   "USDT"}); err != nil && !strings.Contains(err.Error(), "Insufficient BTC balance") {
		t.Errorf("%s PlaceEasyConvert() error %v", ok.Name, err)
	}
}

func TestGetEasyConvertHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetEasyConvertHistory(context.Background(), time.Time{}, time.Time{}, 0); err != nil {
		t.Errorf("%s GetEasyConvertHistory() error %v", ok.Name, err)
	}
}

func TestGetOneClickRepayCurrencyList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetOneClickRepayCurrencyList(context.Background(), "isolated"); err != nil {
		t.Errorf("%s GetOneClickRepayCurrencyList() error %v", ok.Name, err)
	}
}

var oneClickRepayHistoryJSON = `[	{"debtCcy": "ETH","fillDebtSz": "0.01023052","fillRepaySz": "30","repayCcy": "USDT","status": "filled",		"uTime": "1646188520338"	},	{		"debtCcy": "BTC", 		"fillFromSz": "3",		"fillToSz": "60,221.15910001",		"repayCcy": "USDT",		"status": "filled",		"uTime": "1646188520338"	}]`

func TestGetOneClickRepayHistory(t *testing.T) {
	t.Parallel()
	var response []CurrencyOneClickRepay
	if err := json.Unmarshal([]byte(oneClickRepayHistoryJSON), &response); err != nil {
		t.Error("error while deserializing to CurrencyOneClickRepay", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetOneClickRepayHistory(context.Background(), time.Time{}, time.Time{}, 0); err != nil {
		t.Errorf("%s GetOneClickRepayHistory() error %v", ok.Name, err)
	}
}

func TestTradeOneClickRepay(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.TradeOneClickRepay(context.Background(), TradeOneClickRepayParam{
		DebtCurrency:  []string{"BTC"},
		RepayCurrency: "USDT",
	}); err != nil && !strings.Contains(err.Error(), "No permission to use this API") {
		t.Errorf("%s TradeOneClickRepay() error %v", ok.Name, err)
	}
}

func TestGetCounterparties(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetCounterparties(context.Background()); err != nil && !strings.Contains(err.Error(), "No permission to use this API") {
		t.Error("Okx GetCounterparties() error", err)
	}
}

var createRFQInputJSON = `{"anonymous": true,"counterparties":["Trader1","Trader2"],"clRfqId":"rfq01","legs":[{"sz":"25","side":"buy","instId":"BTCUSD-221208-100000-C"},{"sz":"150","side":"buy","instId":"ETH-USDT","tgtCcy":"base_ccy"}]}`
var createRFQOutputJSON = `{"cTime":"1611033737572","uTime":"1611033737572","traderCode":"SATOSHI","rfqId":"22534","clRfqId":"rfq01","state":"active","validUntil":"1611033857557","counterparties":["Trader1","Trader2"],"legs":[{"instId":"BTCUSD-221208-100000-C","sz":"25","side":"buy","tgtCcy":""},{"instId":"ETH-USDT","sz":"150","side":"buy","tgtCcy":"base_ccy"}]}`

func TestCreateRFQ(t *testing.T) {
	t.Parallel()
	var input CreateRFQInput
	if err := json.Unmarshal([]byte(createRFQInputJSON), &input); err != nil {
		t.Error("Okx Decerializing to CreateRFQInput", err)
	}
	var resp RFQResponse
	if err := json.Unmarshal([]byte(createRFQOutputJSON), &resp); err != nil {
		t.Error("Okx Decerializing to CreateRFQResponse", err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, err := ok.CreateRFQ(context.Background(), input); err != nil && !strings.Contains(err.Error(), "No permission to use this API") {
		t.Error("Okx CreateRFQ() error", err)
	}
}

func TestCancelRFQ(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	_, err := ok.CancelRFQ(context.Background(), CancelRFQRequestParam{})
	if err != nil && !errors.Is(err, errMissingRFQIDANDClientSuppliedRFQID) {
		t.Errorf("Okx CancelRFQ() expecting %v, but found %v", errMissingRFQIDANDClientSuppliedRFQID, err)
	}
	_, err = ok.CancelRFQ(context.Background(), CancelRFQRequestParam{
		ClientSuppliedRFQID: "somersdjskfjsdkfj",
	})
	if err != nil && !strings.Contains(err.Error(), "No permission to use this API") {
		t.Error("Okx CancelRFQ() error", err)
	}
}

func TestMultipleCancelRFQ(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	_, err := ok.CancelMultipleRFQs(context.Background(), CancelRFQRequestsParam{})
	if err != nil && !errors.Is(err, errMissingRFQIDANDClientSuppliedRFQID) {
		t.Errorf("Okx CancelMultipleRFQs() expecting %v, but found %v", errMissingRFQIDANDClientSuppliedRFQID, err)
	}
	_, err = ok.CancelMultipleRFQs(context.Background(), CancelRFQRequestsParam{
		ClientSuppliedRFQID: []string{"somersdjskfjsdkfj"},
	})
	if err != nil && !strings.Contains(err.Error(), "Either parameter rfqIds or clRfqIds is required") {
		t.Error("Okx CancelMultipleRFQs() error", err)
	}
}

func TestCancelAllRFQs(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, err := ok.CancelAllRFQs(context.Background()); err != nil &&
		!strings.Contains(err.Error(), "No permission to use this API.") {
		t.Errorf("%s CancelAllRFQs() error %v", ok.Name, err)
	}
}

var executeQuoteJSON = `{"blockTdId":"180184","rfqId":"1419","clRfqId":"r0001","quoteId":"1046","clQuoteId":"q0001","tTraderCode":"Trader1","mTraderCode":"Trader2","cTime":"1649670009","legs":[{	"px":"0.1",	"sz":"25",	"instId":"BTC-USD-20220114-13250-C",	"side":"sell",	"fee":"-1.001",	"feeCcy":"BTC",	"tradeId":"10211"},{	"px":"0.2",	"sz":"25",	"instId":"BTC-USDT",	"side":"buy",	"fee":"-1.001",	"feeCcy":"BTC",	"tradeId":"10212"}]}`

func TestExecuteQuote(t *testing.T) {
	t.Parallel()
	var resp ExecuteQuoteResponse
	if err := json.Unmarshal([]byte(executeQuoteJSON), &resp); err != nil {
		t.Error("Okx Decerialing error", err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, err := ok.ExecuteQuote(context.Background(), ExecuteQuoteParams{}); err != nil && !errors.Is(err, errMissingRfqIDOrQuoteID) {
		t.Errorf("Okx ExecuteQuote() expected %v, but found %v", errMissingRfqIDOrQuoteID, err)
	}
	if _, err := ok.ExecuteQuote(context.Background(), ExecuteQuoteParams{
		RfqID:   "22540",
		QuoteID: "84073",
	}); err != nil && !strings.Contains(err.Error(), "No permission to use this API") {
		t.Error("Okx ExecuteQuote() error", err)
	}
}

func TestSetQuoteProducts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.SetQuoteProducts(context.Background(), []SetQuoteProductParam{
		{
			InstrumentType: "SWAP",
			Data: []MakerInstrumentSetting{
				{
					Underlying:     "BTC-USD",
					MaxBlockSize:   10000,
					MakerPriceBand: 5,
				},
				{
					Underlying: "ETH-USDT",
				},
			},
		}}); err != nil && !strings.Contains(err.Error(), "No permission to use this API") {
		t.Errorf("%s SetQuoteProducts() error %v", ok.Name, err)
	}
}

func TestResetMMPStatus(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.ResetMMPStatus(context.Background()); err != nil && !strings.Contains(err.Error(), "No permission to use this API") {
		t.Errorf("%s ResetMMPStatus() error %v", ok.Name, err)
	}
}

var createQuoteJSON = `{"cTime":"1611038342698","uTime":"1611038342698","quoteId":"84069","clQuoteId":"q002","rfqId":"22537","quoteSide":"buy","state":"active","validUntil":"1611038442838",	"legs":[			{				"px":"39450.0",				"sz":"200000",				"instId":"BTC-USDT-SWAP",				"side":"buy",				"tgtCcy":""			}            	]}`

func TestCreateQuote(t *testing.T) {
	t.Parallel()
	var resp QuoteResponse
	if err := json.Unmarshal([]byte(createQuoteJSON), &resp); err != nil {
		t.Error("Okx Decerializing to CreateQuoteResponse error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.CreateQuote(context.Background(), CreateQuoteParams{}); err != nil && !errors.Is(err, errMissingRfqID) {
		t.Errorf("Okx CreateQuote() expecting %v, but found %v", errMissingRfqID, err)
	}
	if _, err := ok.CreateQuote(context.Background(), CreateQuoteParams{
		RfqID:     "12345",
		QuoteSide: order.Buy,
		Legs: []QuoteLeg{
			{
				Price:          1234,
				SizeOfQuoteLeg: 2,
				InstrumentID:   "SOL-USD-220909",
				Side:           order.Sell,
			},
			{
				Price:          1234,
				SizeOfQuoteLeg: 1,
				InstrumentID:   "SOL-USD-220909",
				Side:           order.Buy,
			},
		},
	}); err != nil && !strings.Contains(err.Error(), "No permission to use this API.") {
		t.Errorf("%s CreateQuote() error %v", ok.Name, err)
	}
}

func TestCancelQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.CancelQuote(context.Background(), CancelQuoteRequestParams{}); err != nil && !errors.Is(err, errMissingQuoteIDOrClientSuppliedQuoteID) {
		t.Error("Okx CancelQuote() error", err)
	}
}

func TestCancelMultipleQuote(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.CancelMultipleQuote(context.Background(), CancelQuotesRequestParams{}); err != nil && !errors.Is(errMissingEitherQuoteIDAOrClientSuppliedQuoteIDs, err) {
		t.Error("Okx CancelQuote() error", err)
	}
}

func TestCancelAllQuotes(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	time, err := ok.CancelAllQuotes(context.Background())
	switch {
	case err != nil &&
		(strings.Contains(err.Error(), "Cancellation failed as you do not have any active Quotes.") ||
			strings.Contains(err.Error(), "No permission to use this API.")):
		t.Skip("Skiping test with reason:", err)
	case err != nil:
		t.Error("Okx CancelAllQuotes() error", err)
	case err == nil && time.IsZero():
		t.Error("Okx CancelAllQuotes() zero timestamp message ")
	}
}

func TestGetRFQs(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetRfqs(context.Background(), &RfqRequestParams{}); err != nil && !strings.Contains(err.Error(), "No permission to use this API.") {
		t.Error("Okx GetRfqs() error", err)
	}
}

func TestGetQuotes(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetQuotes(context.Background(), &QuoteRequestParams{}); err != nil && !strings.Contains(err.Error(), "No permission to use this API") {
		t.Error("Okx GetQuotes() error", err)
	}
}

var rfqTradeResponseJSON = `{
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
	if err := json.Unmarshal([]byte(rfqTradeResponseJSON), &resp); err != nil {
		t.Error("Okx Decerializing to RFQTradeResponse error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetRFQTrades(context.Background(), &RFQTradesRequestParams{}); err != nil && !strings.Contains(err.Error(), "No permission to use this API.") {
		t.Error("Okx GetRFQTrades() error", err)
	}
}

var publicTradesResponseJSON = `{"blockTdId": "439161457415012352","legs": [{"instId": "BTC-USD-210826","side": "sell","sz": "100","px": "11000","tradeId": "439161457415012354"}],"cTime": "1650976251241"}`

func TestGetPublicTrades(t *testing.T) {
	t.Parallel()
	var resp PublicTradesResponse
	if err := json.Unmarshal([]byte(publicTradesResponseJSON), &resp); err != nil {
		t.Error("Okx Decerializing to PublicTradesResponse error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetPublicTrades(context.Background(), "", "", 10); err != nil {
		t.Error("Okx GetPublicTrades() error", err)
	}
}

func TestGetFundingCurrencies(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetFundingCurrencies(context.Background()); err != nil {
		t.Error("Okx  GetFundingCurrencies() error", err)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetBalance(context.Background(), ""); err != nil {
		t.Error("Okx GetBalance() error", err)
	}
}

func TestGetAccountAssetValuation(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetAccountAssetValuation(context.Background(), ""); err != nil {
		t.Error("Okx  GetAccountAssetValuation() error", err)
	}
}

var fundingTransferRequest = `{"ccy":"USDT","type":"4","amt":"1.5","from":"6","to":"6","subAcct":"mini"}`

var fundingTransferResponseMessage = `{"transId": "754147","ccy": "USDT","clientId": "","from": "6","amt": "0.1","to": "18"}`

func TestFundingTransfer(t *testing.T) {
	t.Parallel()
	var fundReq FundingTransferRequestInput
	if err := json.Unmarshal([]byte(fundingTransferRequest), &fundReq); err != nil {
		t.Error("Okx FundingTransferRequestInput{} unmarshal  error", err)
	}
	var fundResponse FundingTransferResponse
	if err := json.Unmarshal([]byte(fundingTransferResponseMessage), &fundResponse); err != nil {
		t.Error("okx FundingTransferRequestInput{} unmarshal error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.FundingTransfer(context.Background(), &FundingTransferRequestInput{
		Amount:   12.000,
		To:       "6",
		From:     "18",
		Currency: "BTC",
	}); err != nil && !strings.Contains(err.Error(), "Insufficient balance") {
		t.Error("Okx FundingTransfer() error", err)
	}
}

var fundingRateTransferResponseJSON = `{"amt": "1.5","ccy": "USDT","clientId": "","from": "18","instId": "","state": "success","subAcct": "test","to": "6","toInstId": "","transId": "1","type": "1"}`

func TestGetFundsTransferState(t *testing.T) {
	t.Parallel()
	var transResponse TransferFundRateResponse
	if err := json.Unmarshal([]byte(fundingRateTransferResponseJSON), &transResponse); err != nil {
		t.Error("Okx TransferFundRateResponse{} unmarshal error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetFundsTransferState(context.Background(), "1", "", 1); err != nil {
		t.Error("Okx GetFundsTransferState() error", err)
	}
}

var assetBillDetailResponse = `{"billId": "12344","ccy": "BTC","clientId": "","balChg": "2","bal": "12","type": "1","ts": "1597026383085"}`

func TestGetAssetBillsDetails(t *testing.T) {
	t.Parallel()
	var response AssetBillDetail
	err := json.Unmarshal([]byte(assetBillDetailResponse), &response)
	if err != nil {
		t.Error("Okx Unmarshaling error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err = ok.GetAssetBillsDetails(context.Background(), "", "", "", time.Time{}, time.Time{}, 0, 5)
	if err != nil {
		t.Error("Okx GetAssetBillsDetail() error", err)
	}
}

var lightningDepositResponseString = `{
	"cTime": "1631171307612",
	"invoice": "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hzzdszuv6yv6yw5svctl8kc8uv6y77szv5kma2kuculj86tk3yys9qyyssqd8urqgcensh9l4zwlwr3lxlcdqrlflvvlwldutm6ljx486h7lylqmd06kky6scas7warx69sregzrx20ffmsr4sp865x3wasrjd8ttgqrlx3tr"
}`

func TestGetLightningDeposits(t *testing.T) {
	t.Parallel()
	var response LightningDepositItem
	err := json.Unmarshal([]byte(lightningDepositResponseString), &response)
	if err != nil {
		t.Error("Okx Unamrshaling to LightningDepositItem error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err = ok.GetLightningDeposits(context.Background(), "BTC", 1.00, 0); err != nil && !strings.Contains(err.Error(), "You have no permission to use this API interface") {
		t.Error("Okx GetLightningDeposits() error", err)
	}
}

var depositAddressResponseItemString = `{"tag":"","chain":"BTC-Bitcoin","ctAddr":"","ccy":"BTC","to":"6","addr":"bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc","selected":true,"memo":"","addrEx":"","pmtId":""}`

func TestGetCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	var response CurrencyDepositResponseItem
	err := json.Unmarshal([]byte(depositAddressResponseItemString), &response)
	if err != nil {
		t.Error("Okx unmarshaling to CurrencyDepositResponseItem error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetCurrencyDepositAddress(context.Background(), "BTC"); err != nil {
		t.Error("Okx GetCurrencyDepositAddress() error", err)
	}
}

var depositHistoryResponseString = `{"amt": "0.01044408","txId": "1915737_3_0_0_asset","ccy": "BTC","chain":"BTC-Bitcoin","from": "13801825426","to": "","ts": "1597026383085","state": "2","depId": "4703879"}`

func TestGetCurrencyDepositHistory(t *testing.T) {
	t.Parallel()
	var response DepositHistoryResponseItem
	err := json.Unmarshal([]byte(depositHistoryResponseString), &response)
	if err != nil {
		t.Error("Okx DepositHistoryResponseItem unmarshaling error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetCurrencyDepositHistory(context.Background(), "BTC", "", "", time.Time{}, time.Time{}, 0, 5); err != nil {
		t.Error("Okx GetCurrencyDepositHistory() error", err)
	}
}

var withdrawalResponseString = `{"amt": "0.1","wdId": "67485","ccy": "BTC","clientId": "","chain": "BTC-Bitcoin"}`

func TestWithdrawal(t *testing.T) {
	t.Parallel()
	var response WithdrawalResponse
	err := json.Unmarshal([]byte(withdrawalResponseString), &response)
	if err != nil {
		t.Error("Okx WithdrawalResponse unmarshaling json error", err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	_, err = ok.Withdrawal(context.Background(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: "BTC", WithdrawalDestination: "4", ToAddress: core.BitcoinDonationAddress})
	if err != nil && !strings.Contains(err.Error(), "Invalid Authority") {
		t.Error("Okx Withdrawal error", err)
	}
}

var lightningWithdrawalResponseJSON = `{"wdId": "121212","cTime": "1597026383085"}`

func TestLightningWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	var response LightningWithdrawalResponse
	if err := json.Unmarshal([]byte(lightningWithdrawalResponseJSON), &response); err != nil {
		t.Error("Okx LightningWithdrawalResponse Json Conversion error ", err)
	}
	_, err := ok.LightningWithdrawal(context.Background(), LightningWithdrawalRequestInput{
		Currency: currency.BTC.String(),
		Invoice:  "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hz",
	})
	if !strings.Contains(err.Error(), `401 raw response: {"msg":"Invalid Authority","code":"50114"}`) {
		t.Error("Okx LightningWithdrawal() error", err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, err := ok.CancelWithdrawal(context.Background(), "fjasdfkjasdk"); err != nil && !strings.Contains(err.Error(), "Invalid Authority") {
		t.Error("Okx CancelWithdrawal() error", err.Error())
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetWithdrawalHistory(context.Background(), "BTC", "", "", "", time.Time{}, time.Time{}, 0, 10); err != nil {
		t.Error("Okx GetWithdrawalHistory() error", err)
	}
}

func TestSmallAssetsConvert(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.SmallAssetsConvert(context.Background(), []string{"BTC", "USDT"}); err != nil && !strings.Contains(err.Error(), "You do not have assets in this currency") {
		t.Error("Okx SmallAssetsConvert() error", err)
	}
}

var savingBalanceResponse = `{"earnings": "0.0010737388791526","redemptAmt": "0.0000000000000000","rate": "0.0100000000000000","ccy": "USDT","amt": "11.0010737453457821","loanAmt": "11.0010630707982819","pendingAmt": "0.0000106745475002"}`

func TestGetSavingBalance(t *testing.T) {
	t.Parallel()
	var resp SavingBalanceResponse
	err := json.Unmarshal([]byte(savingBalanceResponse), &resp)
	if err != nil {
		t.Error("Okx Saving Balance Unmarshaling error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetSavingBalance(context.Background(), "BTC"); err != nil {
		t.Error("Okx GetSavingBalance() error", err)
	}
}

var redemptionOrPurchaseSavingJSON = `{ "ccy":"BTC","amt":"1","side":"purchase","rate": "0.01"}`

func TestSavingsPurchase(t *testing.T) {
	t.Parallel()
	var resp SavingsPurchaseRedemptionResponse
	if err := json.Unmarshal([]byte(redemptionOrPurchaseSavingJSON), &resp); err != nil {
		t.Error("Okx Unmarshaling purchase or redemption error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.SavingsPurchaseOrRedemption(context.Background(), &SavingsPurchaseRedemptionInput{
		Amount:     123.4,
		Currency:   "BTC",
		Rate:       1,
		ActionType: "purchase",
	}); err != nil && !strings.Contains(err.Error(), "Insufficient balance") {
		t.Error("Okx SavingsPurchaseOrRedemption() error", err)
	}
	if _, err := ok.SavingsPurchaseOrRedemption(context.Background(), &SavingsPurchaseRedemptionInput{
		Amount:     123.4,
		Currency:   "BTC",
		Rate:       1,
		ActionType: "redempt",
	}); err != nil && !strings.Contains(err.Error(), "Insufficient balance") {
		t.Error("Okx SavingsPurchaseOrRedemption() error", err)
	}
}

var setLendingRate = `{"ccy": "BTC","rate": "0.02"}`

func TestSetLendingRate(t *testing.T) {
	t.Parallel()
	var resp LendingRate
	if err := json.Unmarshal([]byte(setLendingRate), &resp); err != nil {
		t.Error("Okx Unmarshaling LendingRate error", err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, err := ok.SetLendingRate(context.Background(), LendingRate{Currency: "BTC", Rate: 2}); err != nil &&
		!strings.Contains(err.Error(), "You do not have assets in this currency") {
		t.Error("Okx SetLendingRate() error", err)
	}
}

var lendinghistoryJSON = `{"ccy": "BTC","amt": "0.01","earnings": "0.001","rate": "0.01","ts": "1597026383085"}`

func TestGetLendingHistory(t *testing.T) {
	t.Parallel()
	var res LendingHistory
	if err := json.Unmarshal([]byte(lendinghistoryJSON), &res); err != nil {
		t.Error("Okx Unmarshaling Lending History error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetLendingHistory(context.Background(), "USDT", time.Time{}, time.Time{}, 10); err != nil {
		t.Error("Okx GetLendingHostory() error", err)
	}
}

var publicBorrowInfoJSON = `{"amt":"176040496.75254011","ccy":"USDT","rate":"0.01","ts":"1664215200000"}`

func TestGetPublicBorrowInfo(t *testing.T) {
	t.Parallel()
	var resp PublicBorrowInfo
	if err := json.Unmarshal([]byte(publicBorrowInfoJSON), &resp); err != nil {
		t.Error("Okx Unmarshaling to LendingHistory error", err)
	}
	t.SkipNow()
	if _, err := ok.GetPublicBorrowInfo(context.Background(), ""); err != nil {
		t.Error("Okx GetPublicBorrowInfo() error", err)
	}
}

var convertCurrencyResponseJSON = `{
	"min": "0.0001",
	"max": "0.5",
	"ccy": "BTC"
}`

func TestGetConvertCurrencies(t *testing.T) {
	t.Parallel()
	var resp ConvertCurrency
	if err := json.Unmarshal([]byte(convertCurrencyResponseJSON), &resp); err != nil {
		t.Error("Okx Unmarshaling Json error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetConvertCurrencies(context.Background()); err != nil {
		t.Error("Okx GetConvertCurrencies() error", err)
	}
}

var convertCurrencyPairResponseJSON = `{
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
	if err := json.Unmarshal([]byte(convertCurrencyPairResponseJSON), &resp); err != nil {
		t.Error("Okx Unmarshaling ConvertCurrencyPair error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetConvertCurrencyPair(context.Background(), "USDT", "BTC"); err != nil {
		t.Error("Okx GetConvertCurrencyPair() error", err)
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
	if err := json.Unmarshal([]byte(estimateQuoteJSON), &estimate); err != nil {
		t.Error("Okx Umarshaling EstimateQuoteResponse error", err)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, err := ok.EstimateQuote(context.Background(), &EstimateQuoteRequestInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          "sell",
		RFQAmount:     30,
		RFQSzCurrency: "USDT",
	}); err != nil {
		t.Error("Okx EstimateQuote() error", err)
	}
}

var convertTradeJSONResponse = `{
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
	if err := json.Unmarshal([]byte(convertTradeJSONResponse), &convert); err != nil {
		t.Error("Okx Unmarshaling to ConvertTradeResponse error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.ConvertTrade(context.Background(), &ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          "Buy",
		Size:          2,
		SizeCurrency:  "USDT",
		QuoteID:       "quoterETH-USDT16461885104612381",
	}); err != nil && !strings.Contains(err.Error(), "Service unavailable") {
		t.Error("Okx ConvertTrade() error", err)
	}
}

var convertHistoryResponseJSON = `{
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
	if err := json.Unmarshal([]byte(convertHistoryResponseJSON), &convertHistory); err != nil {
		t.Error("Okx Unmarshaling ConvertHistory error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetConvertHistory(context.Background(), time.Time{}, time.Time{}, 10, ""); err != nil {
		t.Error("Okx GetConvertHistory() error", err)
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
	if err := json.Unmarshal([]byte(accountBalanceInformation), &account); err != nil {
		t.Error("Okx Unmarshaling to account error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetNonZeroBalances(context.Background(), ""); err != nil {
		t.Error("Okx GetBalance() error", err)
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
	if err := json.Unmarshal([]byte(accountPositin), &accountPosition); err != nil {
		t.Error("Okx Unmarshaling to AccountPosition error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetPositions(context.Background(), "", "", ""); err != nil {
		t.Error("Okx GetPositions() error", err)
	}
}

var accountPositionHistoryJSON = `{"cTime": "1654177169995","ccy": "BTC","closeAvgPx": "29786.5999999789081085","closeTotalPos": "1","instId": "BTC-USD-SWAP","instType": "SWAP","lever": "10.0","mgnMode": "cross","openAvgPx": "29783.8999999995535393","openMaxPos": "1","pnl": "0.00000030434156","pnlRatio": "0.000906447858888","posId": "452587086133239818","posSide": "long","triggerPx": "","type": "allClose","uTime": "1654177174419","uly": "BTC-USD"}`

func TestGetPositionsHistory(t *testing.T) {
	t.Parallel()
	var accountHistory AccountPositionHistory
	if err := json.Unmarshal([]byte(accountPositionHistoryJSON), &accountHistory); err != nil {
		t.Error("Okx Unmarshal AccountPositionHistory error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetPositionsHistory(context.Background(), "", "", "", 0, 10, time.Time{}, time.Time{}); err != nil {
		t.Error("Okx GetPositionsHistory() error", err)
	}
}

var accountAndPositionRiskJSON = `{
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
	if err := json.Unmarshal([]byte(accountAndPositionRiskJSON), &accountAndPositionRisk); err != nil {
		t.Error("Okx Decerializing AccountAndPositionRisk error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetAccountAndPositionRisk(context.Background(), ""); err != nil {
		t.Error("Okx GetAccountAndPositionRisk() error", err)
	}
}

func TestGetBillsDetail(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetBillsDetailLast7Days(context.Background(), &BillsDetailQueryParameter{}); err != nil {
		t.Error("Okx GetBillsDetailLast7Days() error", err)
	}
}

func TestGetAccountConfiguration(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetAccountConfiguration(context.Background()); err != nil {
		t.Error("Okx GetAccountConfiguration() error", err)
	}
}

func TestSetPositionMode(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.SetPositionMode(context.Background(), "net_mode"); err != nil {
		t.Error("Okx SetPositionMode() error", err)
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.SetLeverage(context.Background(), SetLeverageInput{
		Currency:     "USDT",
		Leverage:     5,
		MarginMode:   "cross",
		InstrumentID: "BTC-USDT",
	}); err != nil && !errors.Is(err, errNoValidResponseFromServer) && !strings.Contains(err.Error(), "System error, please try again laterr.") {
		t.Error("Okx SetLeverage() error", err)
	}
}

func TestGetMaximumBuySellAmountOROpenAmount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	// Operation is not supported under the current account mode
	if _, err := ok.GetMaximumBuySellAmountOROpenAmount(context.Background(), "BTC-USDT", "cross", "BTC", "", 5); err != nil && !strings.Contains(err.Error(), "51010") {
		t.Error("Okx GetMaximumBuySellAmountOROpenAmount() error", err)
	}
}

func TestGetMaximumAvailableTradableAmount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	// Operation is not supported under the current account mode
	if _, err := ok.GetMaximumAvailableTradableAmount(context.Background(), "BTC-USDT", "BTC", "cross", true, 123); err != nil && !strings.Contains(err.Error(), "51010") {
		t.Error("Okx GetMaximumAvailableTradableAmount() error", err)
	}
}

func TestIncreaseDecreaseMargin(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.IncreaseDecreaseMargin(context.Background(), IncreaseDecreaseMarginInput{
		InstrumentID: "BTC-USDT",
		PositionSide: "long",
		Type:         "add",
		Amount:       1000,
		Currency:     "USD",
	}); err != nil && !strings.Contains(err.Error(), "Unsupported operation") {
		t.Error("Okx IncreaseDecreaseMargin() error", err)
	}
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetLeverage(context.Background(), "BTC-USDT", "cross"); err != nil {
		t.Error("Okx GetLeverage() error", err)
	}
}

func TestGetMaximumLoanOfInstrument(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	// Operation is not supported under the current account mode
	if _, err := ok.GetMaximumLoanOfInstrument(context.Background(), "ZRX-BTC", "isolated", "ZRX"); err != nil && !strings.Contains(err.Error(), "51010") {
		t.Error("Okx GetMaximumLoanOfInstrument() error", err)
	}
}

func TestGetFeeRate(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetTradeFee(context.Background(), "SPOT", "", ""); err != nil {
		t.Error("Okx GetTradeFeeRate() error", err)
	}
}

func TestGetInterestAccruedData(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetInterestAccruedData(context.Background(), 0, 10, "", "", "", time.Time{}, time.Time{}); err != nil {
		t.Error("Okx GetInterestAccruedData() error", err)
	}
}

func TestGetInterestRate(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetInterestRate(context.Background(), ""); err != nil {
		t.Error("Okx GetInterestRate() error", err)
	}
}

func TestSetGreeks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.SetGreeks(context.Background(), "PA"); err != nil {
		t.Error("Okx SetGeeks() error", err)
	}
}

func TestIsolatedMarginTradingSettings(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.IsolatedMarginTradingSettings(context.Background(), IsolatedMode{
		IsoMode:        "autonomy",
		InstrumentType: "MARGIN",
	}); err != nil && !strings.Contains(err.Error(), "51010") { // Operation is not supported under the current account mode
		t.Error("Okx IsolatedMarginTradingSettings() error", err)
	}
}

func TestGetMaximumWithdrawals(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetMaximumWithdrawals(context.Background(), "BTC"); err != nil {
		t.Error("Okx GetMaximumWithdrawals() error", err)
	}
}

func TestGetAccountRiskState(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	// Operation is not supported under the current account mode
	if _, err := ok.GetAccountRiskState(context.Background()); err != nil && !strings.Contains(err.Error(), "51010") {
		t.Error("Okx GetAccountRiskState() error", err)
	}
}

func TestVIPLoansBorrowAndRepay(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.VIPLoansBorrowAndRepay(context.Background(), LoanBorrowAndReplayInput{Currency: "BTC", Side: "borrow", Amount: 12}); err != nil &&
		!strings.Contains(err.Error(), "Your account does not support VIP loan") {
		t.Error("Okx VIPLoansBorrowAndRepay() error", err)
	}
}

func TestGetBorrowAndRepayHistoryForVIPLoans(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetBorrowAndRepayHistoryForVIPLoans(context.Background(), "", time.Time{}, time.Time{}, 12); err != nil {
		t.Error("Okx GetBorrowAndRepayHistoryForVIPLoans() error", err)
	}
}

func TestGetBorrowInterestAndLimit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetBorrowInterestAndLimit(context.Background(), 1, "BTC"); err != nil && !strings.Contains(err.Error(), "59307") { // You are not eligible for VIP loans
		t.Error("Okx GetBorrowInterestAndLimit() error", err)
	}
}

var positionBuilderJSON = `{"imr": "0.005432310199023","mmr": "0.0041787001530946","mr1": "0.0041787001530946","mr2": "0.0000734347499275","mr3": "0","mr4": "0","mr5": "0","mr6": "0.0028031968471","mr7": "0.0022","posData": [	{		"delta": "-0.008926024905498",		"gamma": "-0.0707804093543001",		"instId": "BTC-USD-220325-50000-C",		"instType": "OPTION",		"notionalUsd": "3782.9800000000005",		"pos": "-1",		"theta": "0.000093015207115",		"vega": "-0.0000382697346669"	}],"riskUnit": "BTC-USD","ts": "1646639497536"}`

func TestPositionBuilder(t *testing.T) {
	t.Parallel()
	var resp PositionBuilderResponse
	if err := json.Unmarshal([]byte(positionBuilderJSON), &resp); err != nil {
		t.Error("Okx Decerializing to PositionBuilderResponse error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.PositionBuilder(context.Background(), PositionBuilderInput{
		ImportExistingPosition: true,
	}); err != nil {
		t.Error("Okx PositionBuilder() error", err)
	}
}

func TestGetGreeks(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetGreeks(context.Background(), ""); err != nil && !strings.Contains(err.Error(), "Unsupported operation") {
		t.Error("Okx GetGreeks() error", err)
	}
}

func TestGetPMLimitation(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetPMLimitation(context.Background(), "SWAP", "BTC-USDT"); err != nil {
		t.Errorf("%s GetPMLimitation() error %v", ok.Name, err)
	}
}

// Subaccount endpoint tests.
var subAccountsResponseJSON = `{"enable":true,"subAcct":"test-1","type":"1","label":"trade futures","mobile":"1818181","gAuth":true,"canTransOut": true,"ts":"1597026383085"}`

func TestViewSubaccountList(t *testing.T) {
	t.Parallel()
	var resp SubaccountInfo
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if err := json.Unmarshal([]byte(subAccountsResponseJSON), &resp); err != nil {
		t.Error("Okx Decerializing to SubaccountInfo error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.ViewSubAccountList(context.Background(), true, "", time.Time{}, time.Time{}, 10); err != nil {
		t.Error("Okx ViewSubaccountList() error", err)
	}
}

var resetSubAccountJSON = `{"subAcct": "yongxu",	"label": "v5",	"apiKey": "arg13sdfgs",	"perm": "read,trade",	"ip": "1.1.1.1",	"ts": "1597026383085"}`

func TestResetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	var response SubAccountAPIKeyResponse
	if err := json.Unmarshal([]byte(resetSubAccountJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to SubAccountAPIKeResponse %v", ok.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.ResetSubAccountAPIKey(context.Background(), &SubAccountAPIKeyParam{
		SubAccountName: "samuael",
		APIKey:         apiKey,
	}); err != nil && !strings.Contains(err.Error(), "invalid API Key permission") {
		t.Errorf("%s ResetSubAccountAPIKey() error %v", ok.Name, err)
	}
}

var subaccountBalanceResponseJSON = `{"adjEq": "10679688.0460531643092577","details": [{"availBal": "","availEq": "9930359.9998","cashBal": "9930359.9998","ccy": "USDT","crossLiab": "0","disEq": "9439737.0772999514","eq": "9930359.9998","eqUsd": "9933041.196999946","frozenBal": "0","interest": "0","isoEq": "0","isoLiab": "0","liab": "0","maxLoan": "10000","mgnRatio": "","notionalLever": "","ordFrozen": "0","twap": "0","uTime": "1620722938250","upl": "0","uplLiab": "0"}],"imr": "3372.2942371050594217","isoEq": "0","mgnRatio": "70375.35408747017","mmr": "134.8917694842024","notionalUsd": "33722.9423710505978888","ordFroz": "0","totalEq": "11172992.1657531589092577","uTime": "1623392334718"}`

func TestGetSubaccountTradingBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	var resp SubaccountBalanceResponse
	if err := json.Unmarshal([]byte(subaccountBalanceResponseJSON), &resp); err != nil {
		t.Error("Okx ", err)
	}
	if _, err := ok.GetSubaccountTradingBalance(context.Background(), ""); err != nil && !errors.Is(err, errMissingRequiredParameterSubaccountName) {
		t.Errorf("Okx GetSubaccountTradingBalance() expecting \"%v\", but found \"%v\"", errMissingRequiredParameterSubaccountName, err)
	}
	if _, err := ok.GetSubaccountTradingBalance(context.Background(), "test1"); err != nil && !strings.Contains(err.Error(), "sub-account does not exist") {
		t.Error("Okx GetSubaccountTradingBalance() error", err)
	}
}

var fundingBalanceJSON = `{
	"availBal": "37.11827078",
	"bal": "37.11827078",
	"ccy": "ETH",
	"frozenBal": "0"
}`

func TestGetSubaccountFundingBalance(t *testing.T) {
	t.Parallel()
	var resp FundingBalance
	if err := json.Unmarshal([]byte(fundingBalanceJSON), &resp); err != nil {
		t.Error("okx Decerializing to FundingBalance error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetSubaccountFundingBalance(context.Background(), "test1", ""); err != nil && !strings.Contains(err.Error(), "Sub-account does not exist") {
		t.Error("Okx GetSubaccountFundingBalance() error", err)
	}
}

var historyOfSubaccountTransfer = `{"billId": "12344","type":"1","ccy": "BTC","amt":"2","subAcct":"test-1","ts":"1597026383085"}`

func TestHistoryOfSubaccountTransfer(t *testing.T) {
	t.Parallel()
	var resp SubaccountBillItem
	if err := json.Unmarshal([]byte(historyOfSubaccountTransfer), &resp); err != nil {
		t.Error("Okx Decerializing to SubaccountBillItem error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.HistoryOfSubaccountTransfer(context.Background(), "", 0, "", time.Time{}, time.Time{}, 10); err != nil {
		t.Error("Okx HistoryOfSubaccountTransfer() error", err)
	}
}

func TestMasterAccountsManageTransfersBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(context.Background(), "BTC", 1200, 9, 9, "", "", true); err != nil && !errors.Is(err, errInvalidInvalidSubaccount) {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", err)
	}
	if _, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(context.Background(), "BTC", 1200, 8, 8, "", "", true); err != nil && !errors.Is(err, errInvalidInvalidSubaccount) {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", err)
	}
}

func TestSetPermissionOfTransferOut(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.SetPermissionOfTransferOut(context.Background(), PermissingOfTransfer{SubAcct: "Test1"}); err != nil && !strings.Contains(err.Error(), "Sub-account does not exist") {
		t.Error("Okx SetPermissionOfTransferOut() error", err)
	}
}

func TestGetCustodyTradingSubaccountList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetCustodyTradingSubaccountList(context.Background(), ""); err != nil {
		t.Error("Okx GetCustodyTradingSubaccountList() error", err)
	}
}

var gridTradingPlaceOrder = `{"instId": "BTC-USDT-SWAP","algoOrdType": "contract_grid","maxPx": "5000","minPx": "400","gridNum": "10","runType": "1","sz": "200", "direction": "long","lever": "2"}`

func TestPlaceGridAlgoOrder(t *testing.T) {
	t.Parallel()
	var input GridAlgoOrder
	if err := json.Unmarshal([]byte(gridTradingPlaceOrder), &input); err != nil {
		t.Error("Okx Decerializing to GridALgoOrder error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.PlaceGridAlgoOrder(context.Background(), &input); err != nil {
		t.Error("Okx PlaceGridAlgoOrder() error", err)
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
	if err := json.Unmarshal([]byte(gridOrderAmendAlgo), &input); err != nil {
		t.Error("Okx Decerializing to GridAlgoOrderAmend error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.AmendGridAlgoOrder(context.Background(), input); err != nil {
		t.Error("Okx AmendGridAlgoOrder() error", err)
	}
}

func TestStopGridAlgoOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.StopGridAlgoOrder(context.Background(), []StopGridAlgoOrderRequest{}); err != nil {
		t.Error("Okx StopGridAlgoOrder() error", err)
	}
}

func TestGetGridAlgoOrdersList(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetGridAlgoOrdersList(context.Background(), "grid", "", "", "", "", "", 100); err != nil {
		t.Error("Okx GetGridAlgoOrdersList() error", err)
	}
}

func TestGetGridAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetGridAlgoOrderHistory(context.Background(), "contract_grid", "", "", "", "", "", 100); err != nil {
		t.Error("Okx GetGridAlgoOrderHistory() error", err)
	}
}

func TestGetGridAlgoOrderDetails(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetGridAlgoOrderDetails(context.Background(), "grid", ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderDetails() expecting %v, but found %v error", errMissingAlgoOrderID, err)
	}
	if _, err := ok.GetGridAlgoOrderDetails(context.Background(), "grid", "7878"); err != nil && !strings.Contains(err.Error(), "Order does not exist") {
		t.Error("Okx GetGridAlgoOrderDetails() error", err)
	}
}

func TestGetGridAlgoSubOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetGridAlgoSubOrders(context.Background(), "", "", "", "", "", "", 10); err != nil && !errors.Is(err, errMissingAlgoOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingAlgoOrderType)
	}
	if _, err := ok.GetGridAlgoSubOrders(context.Background(), "grid", "", "", "", "", "", 10); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingAlgoOrderID)
	}
	if _, err := ok.GetGridAlgoSubOrders(context.Background(), "grid", "1234", "", "", "", "", 10); err != nil && !errors.Is(err, errMissingSubOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingSubOrderType)
	}
	if _, err := ok.GetGridAlgoSubOrders(context.Background(), "grid", "1234", "live", "", "", "", 10); err != nil && !errors.Is(err, errMissingSubOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingSubOrderType)
	}
}

var spotGridAlgoOrderPosition = `{"adl": "1","algoId": "449327675342323712","avgPx": "29215.0142857142857149","cTime": "1653400065917","ccy": "USDT","imr": "2045.386","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29206.7","lever": "5","liqPx": "661.1684795867162","markPx": "29213.9","mgnMode": "cross","mgnRatio": "217.19370606167573","mmr": "40.907720000000005","notionalUsd": "10216.70307","pos": "35","posSide": "net","uTime": "1653400066938","upl": "1.674999999999818","uplRatio": "0.0008190504784478"}`

func TestGetGridAlgoOrderPositions(t *testing.T) {
	t.Parallel()
	var resp AlgoOrderPosition
	if err := json.Unmarshal([]byte(spotGridAlgoOrderPosition), &resp); err != nil {
		t.Error("Okx Decerializing to AlgoOrderPosition error", err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetGridAlgoOrderPositions(context.Background(), "", ""); err != nil && !errors.Is(err, errMissingAlgoOrderType) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderType, err)
	}
	if _, err := ok.GetGridAlgoOrderPositions(context.Background(), "contract_grid", ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
	if _, err := ok.GetGridAlgoOrderPositions(context.Background(), "contract_grid", ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
}

func TestSpotGridWithdrawProfit(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.SpotGridWithdrawProfit(context.Background(), ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx SpotGridWithdrawProfit() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
	if _, err := ok.SpotGridWithdrawProfit(context.Background(), "1234"); err != nil && !strings.Contains(err.Error(), "Policy type is not grid policy") {
		t.Skip("Policy type is not grid policy")
	} else if err != nil && !strings.Contains(err.Error(), "The strategy does not exist or has stopped") {
		t.Error("Okx SpotGridWithdrawProfit() error", err)
	}
}

var computeMarginBalanceJSON = `{"lever": "0.3877200981166066","maxAmt": "1.8309562403342999"}`

func TestComputeMarginBalance(t *testing.T) {
	t.Parallel()
	var response ComputeMarginBalance
	if err := json.Unmarshal([]byte(computeMarginBalanceJSON), &response); err != nil {
		t.Errorf("%s ComputeMarginBalance() error %v", ok.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.ComputeMarginBalance(context.Background(), MarginBalanceParam{
		AlgoID: "123456",
		Type:   "other",
	}); err != nil && !errors.Is(err, errInvalidMarginTypeAdjust) {
		t.Errorf("%s ComputeMarginBalance() expected %v, but found %v", ok.Name, errInvalidMarginTypeAdjust, err)
	}
	if _, err := ok.ComputeMarginBalance(context.Background(), MarginBalanceParam{
		AlgoID: "123456",
		Type:   "add",
	}); err != nil && !strings.Contains(err.Error(), "The strategy does not exist or has stopped") {
		t.Errorf("%s ComputeMarginBalance() error %v", ok.Name, err)
	}
}

func TestAdjustMarginBalance(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.AdjustMarginBalance(context.Background(), MarginBalanceParam{
		AlgoID: "1234",
		Type:   "add",
		Amount: 12345,
	}); err != nil && !strings.Contains(err.Error(), "The strategy does not exist or has stopped") {
		t.Errorf("%s AdjustMarginBalance() error %v", ok.Name, err)
	}
}

var gridAIParamJSON = `{"algoOrdType": "grid","annualizedRate": "1.5849","ccy": "USDT","direction": "",	"duration": "7D","gridNum": "5","instId": "BTC-USDT","lever": "0","maxPx": "21373.3","minInvestment": "0.89557758",	"minPx": "15544.2",	"perMaxProfitRate": "0.0733865364573281","perMinProfitRate": "0.0561101403446263","runType": "1"}`

func TestGetGridAIParameter(t *testing.T) {
	t.Parallel()
	var response GridAIParameterResponse
	if err := json.Unmarshal([]byte(gridAIParamJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to GridAIParameterResponse error %v", ok.Name, err)
	}
	if _, err := ok.GetGridAIParameter(context.Background(), "grid", "BTC-USDT", "", ""); err != nil {
		t.Errorf("%s GetGridAIParameter() error %v", ok.Name, err)
	}
}

var getOfferJSON = `{	"ccy": "GLMR",  	"productId":"1234",    	"protocol": "glimmar",	"protocolType":"staking",  	"term":"15",	"apy":"0.5496",	"earlyRedeem":true,	"investData":[	  {		"ccy":"GLMR",		"bal":"100",		"minAmt":"1",		"maxAmt":""	  }	],	"earningData": [	 {		"ccy": "GLMR",		"earningType":"1"	 }	]}`

func TestGetOffers(t *testing.T) {
	t.Parallel()
	var response Offer
	if err := json.Unmarshal([]byte(getOfferJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to Offer %v", ok.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetOffers(context.Background(), "", "", ""); err != nil {
		t.Errorf("%s GetOffers() error %v", ok.Name, err)
	}
}

func TestPurchase(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.Purchase(context.Background(), PurchaseRequestParam{
		ProductID: "1234",
		InvestData: []PurchaseInvestDataItem{
			{
				Currency: "ZIL",
				Amount:   100,
			},
		},
		Term: 30,
	}); err != nil {
		t.Errorf("%s Purchase() %v", ok.Name, err)
	}
}

func TestRedeem(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.Redeem(context.Background(), RedeemRequestParam{
		OrderID:          "754147",
		ProtocolType:     "defi",
		AllowEarlyRedeem: true,
	}); err != nil && !strings.Contains(err.Error(), "Order not found") {
		t.Errorf("%s Redeem() error %v", ok.Name, err)
	}
}

func TestCancelPurchaseOrRedemption(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.CancelPurchaseOrRedemption(context.Background(), CancelFundingParam{
		OrderID:      "754147",
		ProtocolType: "defi",
	}); err != nil && !strings.Contains(err.Error(), "Order not found") {
		t.Errorf("%s CancelPurchaseOrRedemption() error %v", ok.Name, err)
	}
}

var getEarnActiveOrdersJSON = `[{"ordId":"123456","state":"3","ccy": "GLMR","protocol": "glimmar","protocolType":"staking","term":"15", "apy":"0.5496","investData":[{"ccy":"GLMR","amt":"100"}],"earningData": [{"ccy": "GLMR","earningType":"1","realizedEarnings":"3"}],"purchasedTime":"1597026383085","redeemedTime":"1597126383085"},{"ordId":"123457","state":"3","ccy": "USDT","protocol": "compond","protocolType":"defi","term":"0",		 "apy":"0.12",		 "investData":[		   {			 "ccy":"USDT",			 "amt":"20"		   }		 ],		 "earningData": [		  {			 "ccy": "USDT",			 "earningType":"0",			 "realizedEarnings":"3"		  },		  {			 "ccy": "COMP",			 "earningType":"1",			 "realizedEarnings":"3"		  }		 ],		 "purchasedTime":"1597026383085",		 "redeemedTime":"1597126383085"	 },	 {		 "ordId":"123458",		 "state":"3",		 "ccy": "ETH",      		 "protocol": "sushiswap",		 "protocolType":"defi",  		 "term":"0",		 "apy":"0.12",		 "investData":[		   {			 "ccy":"USDT",			 "amt":"100"		   },		   {			 "ccy":"ETH",			 "amt":"0.03"		   }		 ],		 "earningData": [		  {			 "ccy": "SUSHI",			 "earningType":"1" ,			 "realizedEarnings":"3"		  }		 ],		 "purchasedTime":"1597026383085",		 "redeemedTime":"1597126383085"	 },	 {		 "ordId":"123458",		 "state":"3",		 "ccy": "LON",      		 "protocol": "tokenlon",		 "protocolType":"defi",  		 "earningCcy": ["LON"],		 "term":"7",		 "apy":"0.12",		 "investData":[		   {			 "ccy":"LON",			 "amt":"1"		   }		 ],		 "earningData": [		  {			 "ccy": "LON",			 "earningType":"0",			 "realizedEarnings":"3"		  }		 ],		 "purchasedTime":"1597026383085",		 "redeemedTime":"1597126383085"	}]`

func TestGetEarnActiveOrders(t *testing.T) {
	t.Parallel()
	var response []ActiveFundingOrder
	if err := json.Unmarshal([]byte(getEarnActiveOrdersJSON), &response); err != nil {
		t.Errorf("%s error ehile deserializing to ActiveFundingOrder %v", ok.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetEarnActiveOrders(context.Background(), "", "", "", ""); err != nil {
		t.Errorf("%s GetEarnActiveOrders() error %v", ok.Name, err)
	}
}

var fundingOrderHistoryJSON = `[{"ordId":"123456",    "state":"3",    "ccy": "GLMR",       "protocol": "glimmar",    "protocolType":"staking",      "term":"15",    "apy":"0.5496",    "investData":[      {        "ccy":"GLMR",        "amt":"100"      }    ],    "earningData": [     {        "ccy": "GLMR",        "earningType":"1",        "realizedEarnings":"3"     }    ],    "purchasedTime":"1597026383085",    "redeemedTime":"1597126383085"},
    {"ordId":"123457",    "state":"3",    "ccy": "USDT",          "protocol": "compond",     "protocolType":"defi",     "term":"0",    "apy":"0.12",    "investData":[      {        "ccy":"USDT",        "amt":"20"      }    ],    "earningData": [     {        "ccy": "USDT",        "earningType":"0",        "realizedEarnings":"3"     },     {        "ccy": "COMP",        "earningType":"1",        "realizedEarnings":"3"     }    ],    "purchasedTime":"1597026383085",    "redeemedTime":"1597126383085"},
    {"ordId":"123458","state":"3","ccy": "ETH",      "protocol": "sushiswap","protocolType":"defi",  "term":"0","apy":"0.12","investData":[  {    "ccy":"USDT",    "amt":"100"  },  {    "ccy":"ETH",    "amt":"0.03"  }],"earningData": [ {    "ccy": "SUSHI",    "earningType":"1",    "realizedEarnings":"3" }],"purchasedTime":"1597026383085","redeemedTime":"1597126383085"
    },{"ordId":"123458","state":"3","ccy": "LON","protocol": "tokenlon","protocolType":"defi","earningCcy": ["LON"],"term":"7","apy":"0.12","investData":[{"ccy":"LON","amt":"1"}],"earningData": [{"ccy": "LON","earningType":"0","realizedEarnings":"3"}],"purchasedTime":"1597026383085","redeemedTime":"1597126383085"}
]`

func TestGetFundingOrderHistory(t *testing.T) {
	t.Parallel()
	var response []FundingOrder
	if err := json.Unmarshal([]byte(fundingOrderHistoryJSON), &response); err != nil {
		t.Errorf("%s error while deserializing to FundingOrder %v", ok.Name, err)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetFundingOrderHistory(context.Background(), "", "", "", time.Time{}, time.Time{}, 0); err != nil {
		t.Errorf("%s GetFundingOrderHistory() error %v", ok.Name, err)
	}
}

var systemStatusResponseJSON = `{"title": "Spot System Upgrade","state": "scheduled","begin": "1620723600000","end": "1620724200000","href": "","serviceType": "1","system": "classic","scheDesc": ""}`

func TestSystemStatusResponse(t *testing.T) {
	t.Parallel()
	var resp SystemStatusResponse
	if err := json.Unmarshal([]byte(systemStatusResponseJSON), &resp); err != nil {
		t.Error("Okx Deserializing to SystemStatusResponse error", err)
	}
	if _, err := ok.SystemStatusResponse(context.Background(), "completed"); err != nil {
		t.Error("Okx SystemStatusResponse() error", err)
	}
}

/**********************************  Wrapper Functions **************************************/

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	if _, err := ok.FetchTradablePairs(context.Background(), asset.Futures); err != nil {
		t.Error("Okx FetchTradablePairs() error", err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	if err := ok.UpdateTradablePairs(context.Background(), true); err != nil {
		t.Error("Okx UpdateTradablePairs() error", err)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	if _, err := ok.UpdateTicker(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Option); err != nil {
		t.Error("Okx UpdateTicker() error", err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	if err := ok.UpdateTickers(context.Background(), asset.Futures); err != nil {
		t.Error("Okx UpdateTicker() error", err)
	}
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	if _, err := ok.FetchTicker(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.PerpetualSwap); err != nil {
		t.Error("Okx FetchTicker() error", err)
	}
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := ok.FetchOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot); err != nil {
		t.Error("Okx FetchOrderbook() error", err)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	if _, err := ok.UpdateOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot); err != nil {
		t.Error("Okx UpdateOrderbook() error", err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.UpdateAccountInfo(context.Background(), asset.Spot); err != nil {
		t.Error("Okx UpdateAccountInfo() error", err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.FetchAccountInfo(context.Background(), asset.Spot); err != nil {
		t.Error("Okx FetchAccountInfo() error", err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetFundingHistory(context.Background()); err != nil {
		t.Error("Okx GetFundingHistory() error", err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetWithdrawalsHistory(context.Background(), currency.BTC, asset.Spot); err != nil {
		t.Error("Okx GetWithdrawalsHistory() error", err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetRecentTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.PerpetualSwap); err != nil {
		t.Error("Okx GetRecentTrades() error", err)
	}
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	setupWsAuth(t)
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
	setupWsAuth(t)
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
	setupWsAuth(t)
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

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.CancelAllOrders(context.Background(), &order.Cancel{}); err != nil {
		t.Errorf("%s CancelAllOrders() error: %v", ok.Name, err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	_, err := ok.ModifyOrder(context.Background(),
		&order.Modify{
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.LTC, currency.BTC),
			OrderID:   "1234",
			Price:     123456.44,
			Amount:    123,
		})
	if err != nil && !strings.Contains(err.Error(), "Operation failed.") {
		t.Errorf("Okx ModifyOrder() error %v", err)
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
	if _, err := ok.GetDepositAddress(context.Background(), currency.BTC, "", currency.USD.String()); err != nil && !errors.Is(err, errDepositAddressNotFound) {
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
	if _, err := ok.GetPairFromInstrumentID(instruments[0]); err != nil {
		t.Error("Okx GetPairFromInstrumentID() error", err)
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
	pair, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequest = order.GetOrdersRequest{
		Type:      order.Limit,
		Pairs:     currency.Pairs{pair, currency.NewPair(currency.USDT, currency.USD), currency.NewPair(currency.USD, currency.LTC)},
		AssetType: asset.Spot,
	}
	if _, err := ok.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
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
	if _, err := ok.GetOrderHistory(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Okx GetOrderHistory() error", err)
	}
}
func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, err := ok.GetFeeByType(context.Background(), &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	}); err != nil {
		t.Errorf("%s GetFeeByType() error %v", ok.Name, err)
	}
}

func TestValidateCredentials(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if err := ok.ValidateCredentials(context.Background(), asset.Spot); err != nil {
		t.Errorf("%s ValidateCredentials() error %v", ok.Name, err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	startTime := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2021, 2, 15, 0, 0, 0, 0, time.UTC)
	_, err := ok.GetHistoricCandles(context.Background(), pair, asset.Spot, startTime, endTime, kline.Interval(time.Hour*5))
	if err != nil && !strings.Contains(err.Error(), "interval not supported") {
		t.Errorf("Okx GetHistoricCandles() expected %s, but found %v", "interval not supported", err)
	}
	_, err = ok.GetHistoricCandles(context.Background(), pair, asset.Spot, time.Time{}, time.Time{}, kline.Interval(time.Hour*4))
	if err != nil {
		t.Error("Okx GetHistoricCandles() error", err)
	}
}

var wsInstrumentResp = `{"arg": {"channel": "instruments","instType": "FUTURES"},"data": [{"instType": "FUTURES","instId": "BTC-USD-191115","uly": "BTC-USD","category": "1","baseCcy": "","quoteCcy": "","settleCcy": "BTC","ctVal": "10","ctMult": "1","ctValCcy": "USD","optType": "","stk": "","listTime": "","expTime": "","tickSz": "0.01","lotSz": "1","minSz": "1","ctType": "linear","alias": "this_week","state": "live","maxLmtSz":"10000","maxMktSz":"99999","maxTwapSz":"99999","maxIcebergSz":"99999","maxTriggerSz":"9999","maxStopSz":"9999"}]}`

func TestWSInstruments(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(wsInstrumentResp)); err != nil {
		t.Errorf("%s Websocket Instruments Push Data error %v", ok.Name, err)
	}
}

var tickerChannelJSON = `{"arg": {"channel": "tickers","instId": "LTC-USD-200327"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","last": "9999.99","lastSz": "0.1","askPx": "9999.99","askSz": "11","bidPx": "8888.88","bidSz": "5","open24h": "9000","high24h": "10000","low24h": "8888.88","volCcy24h": "2222","vol24h": "2222","sodUtc0": "2222","sodUtc8": "2222","ts": "1597026383085"}]}`

func TestTickerChannel(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(tickerChannelJSON)); err != nil {
		t.Error("Okx TickerChannel push data error", err)
	}
}

var openInterestChannel = `{"arg": {"channel": "open-interest","instId": "LTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","oi": "5000","oiCcy": "555.55","ts": "1597026383085"}]}`

func TestOpenInterestPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(openInterestChannel)); err != nil {
		t.Error("Okx Open Interest Push Data error", err)
	}
}

var candlesticksPushData = `{"arg": {"channel": "candle1D","instId": "BTC-USD-191227"},"data": [["1597026383085","8533.02","8553.74","8527.17","8548.26","45247","529.5858061"]]}`

func TestCandlestickPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(candlesticksPushData)); err != nil {
		t.Error("Okx Candlestick Push Data error", err)
	}
}

var tradePushDataJSON = `{"arg": {"channel": "trades","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","tradeId": "130639474","px": "42219.9","sz": "0.12060306","side": "buy","ts": "1630048897897"}]}`

func TestTradePushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(tradePushDataJSON)); err != nil {
		t.Error("Okx Trade Push Data error", err)
	}
}

var estimatedDeliveryAndExercisePricePushDataJSON = `{"arg": {"args": "estimated-price","instType": "FUTURES","uly": "BTC-USD"},"data": [{"instType": "FUTURES","instId": "BTC-USD-170310","settlePx": "200","ts": "1597026383085"}]}`

func TestEstimatedDeliveryAndExercisePricePushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(estimatedDeliveryAndExercisePricePushDataJSON)); err != nil {
		t.Error("Okx Estimated Delivery and Exercise Price Push Data error", err)
	}
}

var markPricePushData = `{"arg": {"channel": "mark-price","instId": "LTC-USD-190628"},"data": [{"instType": "FUTURES","instId": "LTC-USD-190628","markPx": "0.1","ts": "1597026383085"}]}`

func TestMarkPricePushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(markPricePushData)); err != nil {
		t.Error("Okx Mark Price Push Data error", err)
	}
}

var markPriceCandlestickPushData = `{"arg": {"channel": "mark-price-candle1D","instId": "BTC-USD-190628"},"data": [["1597026383085", "3.721", "3.743", "3.677", "3.708"],["1597026383085", "3.731", "3.799", "3.494", "3.72"]]}`

func TestMarkPriceCandlestickPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(markPriceCandlestickPushData)); err != nil {
		t.Error("Okx Mark Price Candlestick Push Data error", err)
	}
}

var priceLimitPushDataJSON = `{    "arg": {        "channel": "price-limit",        "instId": "LTC-USD-190628"    },    "data": [{        "instId": "LTC-USD-190628",        "buyLmt": "200",        "sellLmt": "300",        "ts": "1597026383085"    }]}`

func TestPriceLimitPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(priceLimitPushDataJSON)); err != nil {
		t.Error("Okx Price Limit Push Data error", err)
	}
}

var testSnapshotOrderbookPushData = `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`
var updateOrderBookPushDataJSON = `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"update","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`

func TestSnapshotAndUpdateOrderBookPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(testSnapshotOrderbookPushData)); err != nil {
		t.Error("Okx Snapshot order book push data error", err)
	}
	if err := ok.WsHandleData([]byte(updateOrderBookPushDataJSON)); err != nil {
		t.Error("Okx Update Order Book Push Data error", err)
	}
}

var snapshotOrderBookPushData = `{"arg":{"channel":"books","instId":"TRX-USD-220812"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`

func TestSnapshotPushData(t *testing.T) {
	if err := ok.WsHandleData([]byte(snapshotOrderBookPushData)); err != nil {
		t.Error("Okx Snapshot order book push data error", err)
	}
}

var calculateOrderbookChecksumUpdateorderbookJSON = `{"Bids":[{"Amount":56,"Price":0.07014,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":608,"Price":0.07011,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":110,"Price":0.07009,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1264,"Price":0.07006,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":2347,"Price":0.07004,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":279,"Price":0.07003,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":52,"Price":0.07001,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":91,"Price":0.06997,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4242,"Price":0.06996,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":486,"Price":0.06995,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":161,"Price":0.06992,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":63,"Price":0.06991,"ID":0,"Period":0,"LiquidationOrders":0,
"OrderCount":0},{"Amount":7518,"Price":0.06988,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":186,"Price":0.06976,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":71,"Price":0.06975,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1086,"Price":0.06973,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":513,"Price":0.06961,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4603,"Price":0.06959,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":186,"Price":0.0695,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3043,"Price":0.06946,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":103,"Price":0.06939,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5053,"Price":0.0693,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5039,"Price":0.06909,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5037,"Price":0.06888,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1526,"Price":0.06886,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5008,"Price":0.06867,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5065,"Price":0.06846,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1572,"Price":0.06826,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1565,"Price":0.06801,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":67,"Price":0.06748,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":111,"Price":0.0674,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10038,"Price":0.0672,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.06652,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1526,"Price":0.06625,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10924,"Price":0.06619,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.05986,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.05387,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.04848,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.04363,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0}],"Asks":[{"Amount":5,"Price":0.07026,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":765,"Price":0.07027,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":110,"Price":0.07028,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1264,"Price":0.0703,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":280,"Price":0.07034,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":2255,"Price":0.07035,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":28,"Price":0.07036,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":63,"Price":0.07037,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":137,"Price":0.07039,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":48,"Price":0.0704,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":32,"Price":0.07041,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3985,"Price":0.07043,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":257,"Price":0.07057,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":7870,"Price":0.07058,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":161,"Price":0.07059,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4539,"Price":0.07061,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1438,"Price":0.07068,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3162,"Price":0.07088,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":99,"Price":0.07104,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5018,"Price":0.07108,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1540,"Price":0.07115,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5080,"Price":0.07129,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1512,"Price":0.07145,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5016,"Price":0.0715,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5026,"Price":0.07171,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5062,"Price":0.07192,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1517,"Price":0.07197,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1511,"Price":0.0726,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10376,"Price":0.07314,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.07354,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10277,"Price":0.07466,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":269,"Price":0.07626,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":269,"Price":0.07636,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.0809,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.08899,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.09789,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.10768,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0}],"Exchange":"Okx","Pair":"BTC-USDT","Asset":"spot","LastUpdated":"0001-01-01T00:00:00Z","LastUpdateID":0,"PriceDuplication":false,"IsFundingRate":false,"RestSnapshot":false,"IDAlignment":false}`

func TestCalculateUpdateOrderbookChecksum(t *testing.T) {
	t.Parallel()
	err := ok.WsHandleData([]byte(snapshotOrderBookPushData))
	if err != nil {
		t.Error("Okx Snapshot order book push data error", err)
	}
	var orderbookBase orderbook.Base
	err = json.Unmarshal([]byte(calculateOrderbookChecksumUpdateorderbookJSON), &orderbookBase)
	if err != nil {
		t.Errorf("%s error while deserializing to orderbook.Base %v", ok.Name, err)
	}
	if err := ok.CalculateUpdateOrderbookChecksum(&orderbookBase, 2832680552); err != nil {
		t.Errorf("%s CalculateUpdateOrderbookChecksum() error: %v", ok.Name, err)
	}
}

var optionSummaryPushDataJSON = `{"arg": {"channel": "opt-summary","uly": "BTC-USD"},"data": [{"instType": "OPTION","instId": "BTC-USD-200103-5500-C","uly": "BTC-USD","delta": "0.7494223636","gamma": "-0.6765419039","theta": "-0.0000809873","vega": "0.0000077307","deltaBS": "0.7494223636","gammaBS": "-0.6765419039","thetaBS": "-0.0000809873","vegaBS": "0.0000077307","realVol": "0","bidVol": "","askVol": "1.5625","markVol": "0.9987","lever": "4.0342","fwdPx": "39016.8143629068452065","ts": "1597026383085"}]}`

func TestOptionSummaryPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(optionSummaryPushDataJSON)); err != nil {
		t.Error("Okx Option Summary Push Data error", err)
	}
}

var fundingRatePushDataJSON = `{"arg": {"channel": "funding-rate","instId": "BTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "BTC-USD-SWAP","fundingRate": "0.018","nextFundingRate": "","fundingTime": "1597026383085"}]}`

func TestFundingRatePushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(fundingRatePushDataJSON)); err != nil {
		t.Error("Okx Funding Rate Push Data error", err)
	}
}

var indexCandlestickPushDataJSON = `{"arg": {"channel": "index-candle30m","instId": "BTC-USD"},"data": [["1597026383085", "3811.31", "3811.31", "3811.31", "3811.31"]]}`

func TestIndexCandlestickPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(indexCandlestickPushDataJSON)); err != nil {
		t.Error("Okx Index Candlestick Push Data error", err)
	}
}

var indexTickerPushDataJSON = `{"arg": {"channel": "index-tickers","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","idxPx": "0.1","high24h": "0.5","low24h": "0.1","open24h": "0.1","sodUtc0": "0.1","sodUtc8": "0.1","ts": "1597026383085"}]}`

func TestIndexTickersPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(indexTickerPushDataJSON)); err != nil {
		t.Error("Okx Index Ticker Push Data error", err)
	}
}

var statusPushDataJSON = `{"arg": {"channel": "status"},"data": [{"title": "Spot System Upgrade","state": "scheduled","begin": "1610019546","href": "","end": "1610019546","serviceType": "1","system": "classic","scheDesc": "","ts": "1597026383085"}]}`

func TestStatusPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(statusPushDataJSON)); err != nil {
		t.Error("Okx Status Push Data error", err)
	}
}

var publicStructBlockTradesPushDataJSON = `{"arg":{"channel":"public-struc-block-trades"},"data":[{"cTime":"1608267227834","blockTdId":"1802896","legs":[{"px":"0.323","sz":"25.0","instId":"BTC-USD-20220114-13250-C","side":"sell","tradeId":"15102"},{"px":"0.666","sz":"25","instId":"BTC-USD-20220114-21125-C","side":"buy","tradeId":"15103"}]}]}`

func TestPublicStructBlockTrades(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(publicStructBlockTradesPushDataJSON)); err != nil {
		t.Error("Okx Public Struct Block Trades error", err)
	}
}

var blockTickerPushDataJSON = `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`

func TestBlockTickerPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(blockTickerPushDataJSON)); err != nil {
		t.Error("Okx Block Tickers push data error", err)
	}
}

var accountPushDataJSON = `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`

func TestAccountPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(accountPushDataJSON)); err != nil {
		t.Error("Okx Account Push Data error", err)
	}
}

var positionPushDataJSON = `{"arg":{"channel":"positions","instType":"FUTURES"},"data":[{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`
var positionPushDataWithUnderlyingJSON = `{"arg": {"channel": "positions","uid": "77982378738415879","instType": "ANY"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`

func TestPositionPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(positionPushDataJSON)); err != nil {
		t.Error("Okx Account Push Data error", err)
	}
	if err := ok.WsHandleData([]byte(positionPushDataWithUnderlyingJSON)); err != nil {
		t.Error("Okx Account Push Data error", err)
	}
}

var balanceAndPositionJSON = `{"arg": {"channel": "balance_and_position","uid": "77982378738415879"},"data": [{"pTime": "1597026383085","eventType": "snapshot","balData": [{"ccy": "BTC","cashBal": "1","uTime": "1597026383085"}],"posData": [{"posId": "1111111111","tradeId": "2","instId": "BTC-USD-191018","instType": "FUTURES","mgnMode": "cross","posSide": "long","pos": "10","ccy": "BTC","posCcy": "","avgPx": "3320","uTIme": "1597026383085"}]}]}`

func TestBalanceAndPosition(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(balanceAndPositionJSON)); err != nil {
		t.Error("Okx Balance And Position error", err)
	}
}

var orderPushDataJSON = `{"arg": {    "channel": "orders",    "instType": "SPOT",    "instId": "BTC-USDT",    "uid": "614488474791936"},"data": [    {        "accFillSz": "0.001",        "amendResult": "",        "avgPx": "31527.1",        "cTime": "1654084334977",        "category": "normal",        "ccy": "",        "clOrdId": "",        "code": "0",        "execType": "M",        "fee": "-0.02522168",        "feeCcy": "USDT",        "fillFee": "-0.02522168",        "fillFeeCcy": "USDT",        "fillNotionalUsd": "31.50818374",        "fillPx": "31527.1",        "fillSz": "0.001",        "fillTime": "1654084353263",        "instId": "BTC-USDT",        "instType": "SPOT",        "lever": "0",        "msg": "",        "notionalUsd": "31.50818374",        "ordId": "452197707845865472",        "ordType": "limit",        "pnl": "0",        "posSide": "",        "px": "31527.1",        "rebate": "0",        "rebateCcy": "BTC",        "reduceOnly": "false",        "reqId": "",        "side": "sell",        "slOrdPx": "",        "slTriggerPx": "",        "slTriggerPxType": "last",        "source": "",        "state": "filled",        "sz": "0.001",        "tag": "",        "tdMode": "cash",        "tgtCcy": "",        "tpOrdPx": "",        "tpTriggerPx": "",        "tpTriggerPxType": "last",        "tradeId": "242589207",        "uTime": "1654084353264"    }]}`

func TestOrderPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(orderPushDataJSON)); err != nil {
		t.Error("Okx Order Push Data error", err)
	}
}

var algoOrdersPushDataJSON = `{"arg": {"channel": "orders-algo","uid": "77982378738415879","instType": "FUTURES","instId": "BTC-USD-200329"},"data": [{"instType": "FUTURES","instId": "BTC-USD-200329","ordId": "312269865356374016","ccy": "BTC","algoId": "1234","px": "999","sz": "3","tdMode": "cross","tgtCcy": "","notionalUsd": "","ordType": "trigger","side": "buy","posSide": "long","state": "live","lever": "20","tpTriggerPx": "","tpTriggerPxType": "","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "","triggerPx": "99","triggerPxType": "last","ordPx": "12","actualSz": "","actualPx": "","tag": "adadadadad","actualSide": "","triggerTime": "1597026383085","cTime": "1597026383000"}]}`

func TestAlgoOrderPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(algoOrdersPushDataJSON)); err != nil {
		t.Error("Okx Algo Order Push Data error", err)
	}
}

var advancedAlgoOrderPushDataJSON = `{"arg":{"channel":"algo-advance","uid": "77982378738415879","instType":"SPOT","instId":"BTC-USDT"},"data":[{"actualPx":"","actualSide":"","actualSz":"0","algoId":"355056228680335360","cTime":"1630924001545","ccy":"","count":"1","instId":"BTC-USDT","instType":"SPOT","lever":"0","notionalUsd":"","ordPx":"","ordType":"iceberg","pTime":"1630924295204","posSide":"net","pxLimit":"10","pxSpread":"1","pxVar":"","side":"buy","slOrdPx":"","slTriggerPx":"","state":"pause","sz":"0.1","szLimit":"0.1","tdMode":"cash","timeInterval":"","tpOrdPx":"","tpTriggerPx":"","tag": "adadadadad","triggerPx":"","triggerTime":"","callbackRatio":"","callbackSpread":"","activePx":"","moveTriggerPx":""}]}`

func TestAdvancedAlgoOrderPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(advancedAlgoOrderPushDataJSON)); err != nil {
		t.Error("Okx Advanced Algo Orders Push Data error", err)
	}
}

var positionRiskPushDataJSON = `{"arg": {"channel": "liquidation-warning","uid": "77982378738415879","instType": "ANY"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`

func TestPositionRiskPushDataJSON(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(positionRiskPushDataJSON)); err != nil {
		t.Error("Okx Position Risk Push Data error", err)
	}
}

var accountGreeksPushData = `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"",    "vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`

func TestAccountGreeksPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(accountGreeksPushData)); err != nil {
		t.Error("Okx Account Greeks Push Data error", err)
	}
}

var rfqsPushDataJSON = `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"",    "vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`

func TestRfqs(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(rfqsPushDataJSON)); err != nil {
		t.Error("Okx RFQS Push Data error", err)
	}
}

var accountsPushDataJSON = `{	"arg": {	  "channel": "account",	  "ccy": "BTC",	  "uid": "77982378738415879"	},	"data": [	  {		"uTime": "1597026383085",		"totalEq": "41624.32",		"isoEq": "3624.32",		"adjEq": "41624.32",		"ordFroz": "0",		"imr": "4162.33",		"mmr": "4",		"notionalUsd": "",		"mgnRatio": "41624.32",		"details": [		  {			"availBal": "",			"availEq": "1",			"ccy": "BTC",			"cashBal": "1",			"uTime": "1617279471503",			"disEq": "50559.01",			"eq": "1",			"eqUsd": "45078.3790756226851775",			"frozenBal": "0",			"interest": "0",			"isoEq": "0",			"liab": "0",			"maxLoan": "",			"mgnRatio": "",			"notionalLever": "0.0022195262185864",			"ordFrozen": "0",			"upl": "0",			"uplLiab": "0",			"crossLiab": "0",			"isoLiab": "0",			"coinUsdPrice": "60000",			"stgyEq":"0",			"spotInUseAmt":"",			"isoUpl":""		  }		]	  }	]}`

func TestAccounts(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(accountsPushDataJSON)); err != nil {
		t.Errorf("%s Accounts push data error %v", ok.Name, err)
	}
}

var quotesPushDataJSON = `{"arg":{"channel":"quotes"},"data":[{"validUntil":"1608997227854","uTime":"1608267227834","cTime":"1608267227834","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-220114-25000-C","side":"sell","tgtCcy":""},{"px":"0.0045","sz":"25","instId":"BTC-USD-220114-35000-C","side":"buy","tgtCcy":""}],"quoteId":"25092","rfqId":"18753","traderCode":"SATS","quoteSide":"sell","state":"canceled","clQuoteId":""}]}`

func TestQuotesPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(quotesPushDataJSON)); err != nil {
		t.Error("Okx Quotes Push Data error", err)
	}
}

var structureBlockTradesPushDataJSON = `{"arg":{"channel":"struc-block-trades"},"data":[{"cTime":"1608267227834","rfqId":"18753","clRfqId":"","quoteId":"25092","clQuoteId":"","blockTdId":"180184","tTraderCode":"ANAND","mTraderCode":"WAGMI","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-20220630-60000-C","side":"sell","fee":"0.1001","feeCcy":"BTC","tradeId":"10211","tgtCcy":""},{"px":"0.0033","sz":"25","instId":"BTC-USD-20220630-50000-C","side":"buy","fee":"0.1001","feeCcy":"BTC","tradeId":"10212","tgtCcy":""}]}]}`

func TestStructureBlockTradesPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(structureBlockTradesPushDataJSON)); err != nil {
		t.Error("Okx Structure Block Trades error", err)
	}
}

var spotGridAlgoOrdersPushDataJSON = `{"arg": {"channel": "grid-orders-spot","instType": "ANY"},"data": [{"algoId": "448965992920907776","algoOrdType": "grid","annualizedRate": "0","arbitrageNum": "0","baseSz": "0","cTime": "1653313834104","cancelType": "0","curBaseSz": "0.001776289214","curQuoteSz": "46.801755866","floatProfit": "-0.4953878967772","gridNum": "6","gridProfit": "0","instId": "BTC-USDC","instType": "SPOT","investment": "100","maxPx": "33444.8","minPx": "24323.5","pTime": "1653476023742","perMaxProfitRate": "0.060375293181491054543","perMinProfitRate": "0.0455275366818586","pnlRatio": "0","quoteSz": "100","runPx": "30478.1","runType": "1","singleAmt": "0.00059261","slTriggerPx": "","state": "running","stopResult": "0","stopType": "0","totalAnnualizedRate": "-0.9643551057262827","totalPnl": "-0.4953878967772","tpTriggerPx": "","tradeNum": "3","triggerTime": "1653378736894","uTime": "1653378736894"}]}`

func TestSpotGridAlgoOrdersPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(spotGridAlgoOrdersPushDataJSON)); err != nil {
		t.Error("Okx Spot Grid Algo Orders Push Data error", err)
	}
}

var contractGridAlgoOrdersPushDataJSON = `{"arg": {"channel": "grid-orders-contract","instType": "ANY"},"data": [{"actualLever": "1.02","algoId": "449327675342323712","algoOrdType": "contract_grid","annualizedRate": "0.7572437878956523","arbitrageNum": "1","basePos": true,"cTime": "1653400065912","cancelType": "0","direction": "long","eq": "10129.419829834853","floatProfit": "109.537858234853","gridNum": "50","gridProfit": "19.8819716","instId": "BTC-USDT-SWAP","instType": "SWAP","investment": "10000","lever": "5","liqPx": "603.2149534767834","maxPx": "100000","minPx": "10","pTime": "1653484573918","perMaxProfitRate": "995.7080916791230692","perMinProfitRate": "0.0946277854875634","pnlRatio": "0.0129419829834853","runPx": "29216.3","runType": "1","singleAmt": "1","slTriggerPx": "","state": "running","stopType": "0","sz": "10000","tag": "","totalAnnualizedRate": "4.929207431970923","totalPnl": "129.419829834853","tpTriggerPx": "","tradeNum": "37","triggerTime": "1653400066940","uTime": "1653484573589","uly": "BTC-USDT"}]}`

func TestContractGridAlgoOrdersPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(contractGridAlgoOrdersPushDataJSON)); err != nil {
		t.Error("Okx Contract Grid Algo Order Push Data error", err)
	}
}

var gridPositionsPushDataJSON = `{"arg": {"channel": "grid-positions","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"adl": "1","algoId": "449327675342323712","avgPx": "29181.4638888888888895","cTime": "1653400065917","ccy": "USDT","imr": "2089.2690000000002","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29852.7","lever": "5","liqPx": "604.7617536513744","markPx": "29849.7","mgnMode": "cross","mgnRatio": "217.71740878394456","mmr": "41.78538","notionalUsd": "10435.794191550001","pTime": "1653536068723","pos": "35","posSide": "net","uTime": "1653445498682","upl": "232.83263888888962","uplRatio": "0.1139826489932205"}]}`

func TestGridPositionsPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(gridPositionsPushDataJSON)); err != nil {
		t.Error("Okx Grid Positions Push Data error", err)
	}
}

var gridSubOrdersPushDataJSON = `{"arg": {"channel": "grid-sub-orders","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"accFillSz": "0","algoId": "449327675342323712","algoOrdType": "contract_grid","avgPx": "0","cTime": "1653445498664","ctVal": "0.01","fee": "0","feeCcy": "USDT","groupId": "-1","instId": "BTC-USDT-SWAP","instType": "SWAP","lever": "5","ordId": "449518234142904321","ordType": "limit","pTime": "1653486524502","pnl": "","posSide": "net","px": "28007.2","side": "buy","state": "live","sz": "1","tag":"","tdMode": "cross","uTime": "1653445498674"}]}`

func TestGridSubOrdersPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(gridSubOrdersPushDataJSON)); err != nil {
		t.Error("Okx Grid Sub orders Push Data error", err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetHistoricTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot, time.Time{}, time.Time{}); err != nil {
		t.Errorf("%s GetHistoricTrades() error %v", ok.Name, err)
	}
}

func setupWsAuth(t *testing.T) {
	t.Helper()
	wsSetupLocker.Lock()
	if wsSetupRan {
		wsSetupLocker.Unlock()
		return
	}
	var err error
	if !ok.Websocket.IsEnabled() &&
		!canManipulateRealOrders {
		t.Skip(stream.WebsocketNotEnabled)
	}
	var dialer websocket.Dialer
	err = ok.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		t.Fatal(err)
	}
	ok.Websocket.Wg.Add(2)
	ok.WsResponseMultiplexer = wsRequestDataChannelsMultiplexer{
		WsResponseChannelsMap: make(map[string]*wsRequestInfo),
		Register:              make(chan *wsRequestInfo),
		Unregister:            make(chan string),
		Message:               make(chan *wsIncomingData),
	}
	go ok.WsResponseMultiplexer.Run()
	go ok.wsFunnelConnectionData(ok.Websocket.Conn)
	go ok.WsReadData()
	if ok.IsWebsocketAuthenticationSupported() {
		var authDialer websocket.Dialer
		authDialer.ReadBufferSize = 8192
		authDialer.WriteBufferSize = 8192
		err = ok.WsAuth(context.TODO(), &authDialer)
		if err != nil {
			ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}
	wsSetupRan = true
	wsSetupLocker.Unlock()
}

// ************************** Public Channel Subscriptions *****************************

func TestInstrumentsSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.InstrumentsSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s InstrumentsSubscription() error: %v", ok.Name, err)
	}
}

func TestTickersSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.TickersSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s TickersSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.TickersSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s TickersSubscription() error: %v", ok.Name, err)
	}
}
func TestOpenInterestSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.OpenInterestSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.BTC, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s OpenInterestSubscription() error: %v", ok.Name, err)
	}
}
func TestCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	futuresPairs, err := ok.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Errorf("%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	}
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	currencyPair, err := currency.NewPairFromString(futuresPairs[0])
	if err != nil {
		t.Error(err)
	}
	if _, err := ok.CandlesticksSubscription("subscribe", okxChannelCandle1m, asset.Futures, currencyPair); err != nil {
		t.Errorf("%s CandlesticksSubscription() error: %v", ok.Name, err)
	}
}

func TestTradesSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.TradesSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s TradesSubscription() error: %v", ok.Name, err)
	}
}

func TestEstimatedDeliveryExercisePriceSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	futuresPairs, err := ok.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Errorf("%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	}
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	currencyPair, err := currency.NewPairFromString(futuresPairs[0])
	if err != nil {
		t.Error(err)
	}
	if _, err := ok.EstimatedDeliveryExercisePriceSubscription("subscribe", asset.Futures, currencyPair); err != nil {
		t.Errorf("%s EstimatedDeliveryExercisePriceSubscription() error: %v", ok.Name, err)
	}
}

func TestMarkPriceSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	futuresPairs, err := ok.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Errorf("%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	}
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	currencyPair, err := currency.NewPairFromString(futuresPairs[0])
	if err != nil {
		t.Error(err)
	}
	if _, err := ok.MarkPriceSubscription("subscribe", asset.Futures, currencyPair); err != nil {
		t.Errorf("%s MarkPriceSubscription() error: %v", ok.Name, err)
	}
}

func TestMarkPriceCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	futuresPairs, err := ok.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Errorf("%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	}
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	currencyPair, err := currency.NewPairFromString(futuresPairs[0])
	if err != nil {
		t.Error(err)
	}
	if _, err := ok.MarkPriceSubscription("subscribe", asset.Futures, currencyPair); err != nil {
		t.Errorf("%s MarkPriceSubscription() error: %v", ok.Name, err)
	}
}

func TestPriceLimitSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.PriceLimitSubscription("subscribe", "BTC-USDT-SWAP"); err != nil {
		t.Errorf("%s PriceLimitSubscription() error: %v", ok.Name, err)
	}
}

func TestOrderBooksSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	futuresPairs, err := ok.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Errorf("%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	}
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	currencyPair, err := currency.NewPairFromString(futuresPairs[0])
	if err != nil {
		t.Error(err)
	}
	if _, err := ok.OrderBooksSubscription("subscribe", okxChannelOrderBooks, asset.Futures, currencyPair); err != nil {
		t.Errorf("%s OrderBooksSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.OrderBooksSubscription("unsubscribe", okxChannelOrderBooks, asset.Futures, currencyPair); err != nil {
		t.Errorf("%s OrderBooksSubscription() error: %v", ok.Name, err)
	}
}

func TestOptionSummarySubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.OptionSummarySubscription("subscribe", currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s OptionSummarySubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.OptionSummarySubscription("unsubscribe", currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s OptionSummarySubscription() error: %v", ok.Name, err)
	}
}

func TestFundingRateSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.FundingRateSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP"))); err != nil {
		t.Errorf("%s FundingRateSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.FundingRateSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP"))); err != nil {
		t.Errorf("%s FundingRateSubscription() error: %v", ok.Name, err)
	}
}

func TestIndexCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.IndexCandlesticksSubscription("subscribe", okxChannelIndexCandle6M, asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s IndexCandlesticksSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.IndexCandlesticksSubscription("unsubscribe", okxChannelIndexCandle6M, asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s IndexCandlesticksSubscription() error: %v", ok.Name, err)
	}
}
func TestIndexTickerChannelIndexTickerChannel(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.IndexTickerChannel("subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s IndexTickerChannel() error: %v", ok.Name, err)
	}
	if _, err := ok.IndexTickerChannel("unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s IndexTickerChannel() error: %v", ok.Name, err)
	}
}

func TestStatusSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.StatusSubscription("subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s StatusSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.StatusSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s StatusSubscription() error: %v", ok.Name, err)
	}
}

func TestPublicStructureBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.PublicStructureBlockTradesSubscription("subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s PublicStructureBlockTradesSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.PublicStructureBlockTradesSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s PublicStructureBlockTradesSubscription() error: %v", ok.Name, err)
	}
}
func TestBlockTickerSubscription(t *testing.T) {
	t.Parallel()
	setupWsAuth(t)
	if _, err := ok.BlockTickerSubscription("subscribe", asset.Option, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s BlockTickerSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.BlockTickerSubscription("unsubscribe", asset.Option, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s BlockTickerSubscription() error: %v", ok.Name, err)
	}
}

// ************ Authenticated Websocket endpoints Test **********************************************

func TestWsAccountSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.WsAccountSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s WsAccountSubscription() error: %v", ok.Name, err)
	}
}

func TestWsPlaceOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.WsPlaceOrder(&PlaceOrderRequestParam{
		ClientSupplierOrderID: "my-new-id",
		InstrumentID:          "BTC-USDC",
		TradeMode:             "cross",
		Side:                  "buy",
		OrderType:             "limit",
		QuantityToBuyOrSell:   2.6,
		Price:                 2.1,
	}); err != nil {
		t.Errorf("%s WsPlaceOrder() error: %v", ok.Name, err)
	}
}

func TestWsPlaceMultipleOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.WsPlaceMultipleOrder([]PlaceOrderRequestParam{
		{
			InstrumentID:        "GNX-BTC",
			TradeMode:           "cross",
			Side:                "sell",
			OrderType:           "limit",
			QuantityToBuyOrSell: 1,
			Price:               1,
		},
	}); err != nil {
		t.Error("Okx WsPlaceMultipleOrder() error", err)
	}
}

func TestWsCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.WsCancelOrder(CancelOrderRequestParam{
		InstrumentID: "BTC-USD-190927",
		OrderID:      "2510789768709120",
	}); err != nil && !strings.Contains(err.Error(), "order does not exist.") {
		t.Error("Okx WsCancelOrder() error", err)
	}
}

func TestWsCancleMultipleOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.WsCancelMultipleOrder([]CancelOrderRequestParam{{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
	}}); err != nil {
		t.Error("Okx WsCancleMultipleOrder() error", err)
	}
}

func TestWsAmendOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.WsAmendOrder(&AmendOrderRequestParams{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
		NewQuantity:  1234,
	}); err != nil && !strings.Contains(err.Error(), "order does not exist.") {
		t.Errorf("%s WsAmendOrder() error %v", ok.Name, err)
	}
}

func TestWsAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.WsAmendMultipleOrders([]AmendOrderRequestParams{
		{
			InstrumentID: "DCR-BTC",
			OrderID:      "2510789768709120",
			NewPrice:     1233324.332,
			NewQuantity:  1234,
		},
	}); err != nil && !strings.Contains(err.Error(), "order does not exist.") {
		t.Errorf("%s WsAmendMultipleOrders() %v", ok.Name, err)
	}
}

func TestWsPositionChannel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.WsPositionChannel("subscribe", asset.Option, currency.NewPair(currency.USD, currency.BTC)); err != nil {
		t.Errorf("%s WsPositionChannel() error : %v", ok.Name, err)
	}
}

func TestBalanceAndPositionSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.BalanceAndPositionSubscription("subscribe", "1234"); err != nil {
		t.Errorf("%s BalanceAndPositionSubscription() error %v", ok.Name, err)
	}
	if _, err := ok.BalanceAndPositionSubscription("unsubscribe", "1234"); err != nil {
		t.Errorf("%s BalanceAndPositionSubscription() error %v", ok.Name, err)
	}
}

func TestWsOrderChannel(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.WsOrderChannel("subscribe", asset.Margin, currency.NewPair(currency.SOL, currency.USDT), ""); err != nil {
		t.Errorf("%s WsOrderChannel() error: %v", ok.Name, err)
	}
	if _, err := ok.WsOrderChannel("unsubscribe", asset.Margin, currency.NewPair(currency.SOL, currency.USDT), ""); err != nil {
		t.Errorf("%s WsOrderChannel() error: %v", ok.Name, err)
	}
}

func TestAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.AlgoOrdersSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s AlgoOrdersSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.AlgoOrdersSubscription("unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s AlgoOrdersSubscription() error: %v", ok.Name, err)
	}
}

func TestAdvanceAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.AdvanceAlgoOrdersSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")), ""); err != nil {
		t.Errorf("%s AdvanceAlgoOrdersSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.AdvanceAlgoOrdersSubscription("unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")), ""); err != nil {
		t.Errorf("%s AdvanceAlgoOrdersSubscription() error: %v", ok.Name, err)
	}
}

func TestPositionRiskWarningSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.PositionRiskWarningSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s PositionRiskWarningSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.PositionRiskWarningSubscription("unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s PositionRiskWarningSubscription() error: %v", ok.Name, err)
	}
}

func TestAccountGreeksSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.AccountGreeksSubscription("subscribe", currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s AccountGreeksSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.AccountGreeksSubscription("unsubscribe", currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s AccountGreeksSubscription() error: %v", ok.Name, err)
	}
}

func TestRfqSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.RfqSubscription("subscribe", ""); err != nil {
		t.Errorf("%s RfqSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.RfqSubscription("unsubscribe", ""); err != nil {
		t.Errorf("%s RfqSubscription() error: %v", ok.Name, err)
	}
}

func TestQuotesSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.QuotesSubscription("subscribe"); err != nil {
		t.Errorf("%s QuotesSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.QuotesSubscription("unsubscribe"); err != nil {
		t.Errorf("%s QuotesSubscription() error: %v", ok.Name, err)
	}
}

func TestStructureBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.StructureBlockTradesSubscription("subscribe"); err != nil {
		t.Errorf("%s StructureBlockTradesSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.StructureBlockTradesSubscription("unsubscribe"); err != nil {
		t.Errorf("%s StructureBlockTradesSubscription() error: %v", ok.Name, err)
	}
}

func TestSpotGridAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.SpotGridAlgoOrdersSubscription("subscribe", asset.Empty, currency.EMPTYPAIR, ""); err != nil {
		t.Errorf("%s SpotGridAlgoOrdersSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.SpotGridAlgoOrdersSubscription("unsubscribe", asset.Empty, currency.EMPTYPAIR, ""); err != nil {
		t.Errorf("%s SpotGridAlgoOrdersSubscription() error: %v", ok.Name, err)
	}
}

func TestContractGridAlgoOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.ContractGridAlgoOrders("subscribe", asset.Empty, currency.EMPTYPAIR, ""); err != nil {
		t.Errorf("%s ContractGridAlgoOrders() error: %v", ok.Name, err)
	}
	if _, err := ok.ContractGridAlgoOrders("unsubscribe", asset.Empty, currency.EMPTYPAIR, ""); err != nil {
		t.Errorf("%s ContractGridAlgoOrders() error: %v", ok.Name, err)
	}
}

func TestGridPositionsSubscription(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.GridPositionsSubscription("subscribe", "1234"); err != nil {
		t.Errorf("%s GridPositionsSubscription() error: %v", ok.Name, err)
	}
	if _, err := ok.GridPositionsSubscription("unsubscribe", "1234"); err != nil {
		t.Errorf("%s GridPositionsSubscription() error: %v", ok.Name, err)
	}
}

func TestGridSubOrders(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	setupWsAuth(t)
	if _, err := ok.GridSubOrders("subscribe", ""); err != nil {
		t.Errorf("%s GridSubOrders() error: %v", ok.Name, err)
	}
	if _, err := ok.GridSubOrders("unsubscribe", ""); err != nil {
		t.Errorf("%s GridSubOrders() error: %v", ok.Name, err)
	}
}
