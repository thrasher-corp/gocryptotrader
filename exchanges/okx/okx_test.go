package okx

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
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
	_, er := ok.GetInstruments(context.Background(), &InstrumentsFetchParams{
		InstrumentType: "SPOT",
	})
	if er != nil {
		t.Error("Okx GetInstruments() error", er)
	}
}

// TODO: this handler function has  to be amended
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
