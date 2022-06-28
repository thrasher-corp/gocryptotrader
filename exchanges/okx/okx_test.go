package okx

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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
	ok.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	exchCfg, err := cfg.GetExchangeConfig("okx")
	if err != nil {
		log.Fatal(err)
	}
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.ClientID = passphrase

	err = ok.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func areTestAPIKeysSet() bool {
	return ok.ValidateAPICredentials(ok.GetDefaultCredentials()) == nil
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, er := ok.GetTickers(context.Background(), "SPOT", "", "BTC-USD-SWAP")
	if er != nil {
		t.Error("Okx GetTickers() error", er)
	}
}

func TestGetIndexTickers(t *testing.T) {
	t.Parallel()
	_, er := ok.GetIndexTickers(context.Background(), "USDT", "")
	if er != nil {
		t.Error("OKX GetIndexTickers() error", er)
	}
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, er := ok.GetOrderBookDepth(context.Background(), currency.NewPair(currency.BTC, currency.USDT), 10)
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
	_, er := ok.GetIndexComponents(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	if er != nil {
		t.Error("Okx GetIndexComponents() error", er)
	}
}

func TestGetinstrument(t *testing.T) {
	t.Parallel()
	instruments, er := ok.GetInstruments(context.Background(), &InstrumentsFetchParams{
		InstrumentType: "MARGIN",
	})
	if er != nil {
		t.Error("Okx GetInstruments() error", er)
	}
	val, _ := json.Marshal(instruments)
	println(string(val))
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
	t.Skip()
	var repo DeliveryHistoryResponse
	if err := json.Unmarshal([]byte(deliveryHistoryData), &repo); err != nil {
		t.Error("Okx error", err)
	}
	_, er := ok.GetDeliveryHistory(context.Background(), "FUTURES", "FUTURES", time.Time{}, time.Time{}, 100)
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
	if resp, er := ok.PlaceOrder(context.Background(), PlaceOrderRequestParam{
		InstrumentID:        "MATIC-USDC",
		TradeMode:           "cross",
		Side:                "sell",
		OrderType:           "optimal_limit_ioc",
		QuantityToBuyOrSell: 1,
		OrderPrice:          1,
	}); er != nil {
		t.Error("Okx PlaceOrder() error", er)
	} else {
		binary, _ := json.Marshal(resp)
		println(string(binary))
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

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.CancelOrder(context.Background(),
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
	// this test function has to be re-modified
	if _, er := ok.ClosePositions(context.Background(), &ClosePositionsRequestParams{
		InstrumentID: "BTC-USDT",
		MarginMode:   "cross",
		PositionSide: "long",
		Currency:     currency.BTC.String(),
	}); !strings.Contains(er.Error(), "Token does not exist.") {
		t.Error("Okc ClosePositions() error", er)
	}
}

var orderDetail = `{
	"instType": "FUTURES",
	"instId": "BTC-USD-200329",
	"ccy": "",
	"ordId": "312269865356374016",
	"clOrdId": "b1",
	"tag": "",
	"px": "999",
	"sz": "3",
	"pnl": "5",
	"ordType": "limit",
	"side": "buy",
	"posSide": "long",
	"tdMode": "isolated",
	"accFillSz": "0",
	"fillPx": "0",
	"tradeId": "0",
	"fillSz": "0",
	"fillTime": "0",
	"state": "live",
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
	"rebate": "",
	"tgtCcy":"",
	"category": "",
	"uTime": "1597026383085",
	"cTime": "1597026383085"
  }`

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
	}); !strings.Contains(er.Error(), "Instrument ID does not exist.") {
		t.Error("Okx GetOrderDetail() error", er)
	}
}

const pendingOrderItemJSON = `{
	"accFillSz": "0",
	"avgPx": "",
	"cTime": "1618235248028",
	"category": "normal",
	"ccy": "",
	"clOrdId": "",
	"fee": "0",
	"feeCcy": "BTC",
	"fillPx": "",
	"fillSz": "0",
	"fillTime": "",
	"instId": "BTC-USDT",
	"instType": "SPOT",
	"lever": "5.6",
	"ordId": "301835739059335168",
	"ordType": "limit",
	"pnl": "0",
	"posSide": "net",
	"px": "59200",
	"rebate": "0",
	"rebateCcy": "USDT",
	"side": "buy",
	"slOrdPx": "",
	"slTriggerPx": "",
	"slTriggerPxType": "last",
	"state": "live",
	"sz": "1",
	"tag": "",
	"tgtCcy": "",
	"tdMode": "cross",
	"source":"",
	"tpOrdPx": "",
	"tpTriggerPx": "",
	"tpTriggerPxType": "last",
	"tradeId": "",
	"uTime": "1618235248028"
}`

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
	if _, er = ok.GetCurrencyDepositAddress(context.Background(), "BTC"); er != nil {
		t.Error("Okx GetCurrencyDepositAddress() error", er)
	}
}

var depositHistoryResponseString = `{
	"amt": "0.01044408",
	"txId": "1915737_3_0_0_asset",
	"ccy": "BTC",
	"chain":"BTC-Bitcoin",
	"from": "13801825426",
	"to": "",
	"ts": "1597026383085",
	"state": "2",
	"depId": "4703879"
  }`

func TestGetCurrencyDepositHistory(t *testing.T) {
	t.Parallel()
	var response DepositHistoryResponseItem
	er := json.Unmarshal([]byte(depositHistoryResponseString), &response)
	if er != nil {
		t.Error("Okx DepositHistoryResponseItem unmarshaling error", er)
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
		// || !canManipulateRealOrders
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

var setLendingRate = `{
	"ccy": "BTC",
	"rate": "0.02"
}`

func TestSetLendingRate(t *testing.T) {
	t.Parallel()
	var resp LendingRate
	if er := json.Unmarshal([]byte(setLendingRate), &resp); er != nil {
		t.Error("Okx Unmarshaling LendingRate error", er)
	}
	if !areTestAPIKeysSet() || !canManipulateRealOrders {
		t.SkipNow()
	}
	if _, er := ok.SetLendingRate(context.Background(), LendingRate{Currency: "BTC", Rate: 2}); er != nil {
		t.Error("Okx SetLendingRate() error", er)
	}
}

var lendinghistoryJSON = `{
	"ccy": "BTC",
	"amt": "0.01",
	"earnings": "0.001",
	"rate": "0.01",
	"ts": "1597026383085"
}`

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
		Side:          order.Buy,
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
	// By default, because of the input I used in this method call, this test returns an error message of no valid response from the server.
	// so, I am catching other error messages.
	if _, er := ok.ConvertTrade(context.Background(), ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          order.Buy,
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
	// for this test, there is no response from the server and i am handling the response with errNoValidResponseFromServer
	if _, er := ok.SetLeverage(context.Background(), SetLeverageInput{
		Leverage:     "30",
		MarginMode:   "cross",
		InstrumentID: "",
		Currency:     "BTC",
		PositionSide: "long",
	}); er != nil && !errors.Is(er, errNoValidResponseFromServer) {
		t.Error("Okx SetLeverage() error", er)
	}
}

func TestGetMaximumBuySellAmountOROpenAmount(t *testing.T) {
	t.Parallel()
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.GetMaximumBuySellAmountOROpenAmount(context.Background(), "BTC-USDT", "cross", "BTC", "", 0); er != nil {
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
	if _, er := ok.GetTradeFeeRate(context.Background(), "SPOT", "", ""); er != nil {
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
		IsoMode:        "automatic",
		InstrumentType: "CONTRACTS",
	}); er != nil {
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
	if _, er := ok.VIPLoansBorrowAndRepay(context.Background(), LoanBorrowAndReplayInput{Currency: "BTC", Side: "borrow", Amount: 12}); er != nil {
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
	if _, er := ok.PositionBuilder(context.Background(), PositionBuilderInput{}); er != nil {
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
	var resp RFQCreateResponse
	if er := json.Unmarshal([]byte(createRFQOutputJson), &resp); er != nil {
		t.Error("Okx Decerializing to CreateRFQResponse", er)
	}
	if !areTestAPIKeysSet() {
		t.SkipNow()
	}
	if _, er := ok.CreateRFQ(context.Background(), input); er != nil {
		t.Error("Okx CreateRFQ() error", er)
	}
}
