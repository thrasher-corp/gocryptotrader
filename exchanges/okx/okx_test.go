package okx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
	useTestNet              = false
)

var ok = &Okx{}

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
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.ClientID = passphrase
	ok.SetDefaults()
	if apiKey != "" && apiSecret != "" && passphrase != "" {
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
	}
	if !useTestNet {
		ok.Websocket = sharedtestvalues.NewTestWebsocket()
	}
	err = ok.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	request.MaxRequestJobs = 200
	if !useTestNet {
		ok.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
		ok.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
		setupWS()
	}
	err = ok.UpdateTradablePairs(contextGenerate(), true)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

// contextGenerate sends an optional value to allow test requests
// named this way, so it shows up in auto-complete and reminds you to use it
func contextGenerate() context.Context {
	ctx := context.Background()
	if useTestNet {
		ctx = context.WithValue(ctx, testNetKey("testnet"), useTestNet)
	}
	return ctx
}

func TestStart(t *testing.T) {
	t.Parallel()
	err := ok.Start(context.Background(), nil)
	if !errors.Is(err, common.ErrNilPointer) {
		t.Fatalf("received: '%v' but expected: '%v'", err, common.ErrNilPointer)
	}
	var testWg sync.WaitGroup
	err = ok.Start(context.Background(), &testWg)
	if err != nil {
		t.Fatal(err)
	}
	testWg.Wait()
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTickers(context.Background(), "OPTION", "", "SOL-USD")
	if err != nil {
		t.Error("Okx GetTickers() error", err)
	}
}

func TestGetIndexTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.GetIndexTickers(context.Background(), "USDT", "NEAR-USDT-SWAP")
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
	_, err := ok.GetOrderBookDepth(context.Background(), "BTC-USDT", 400)
	if err != nil {
		t.Error("OKX GetOrderBookDepth() error", err)
	}
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCandlesticks(context.Background(), "BTC-USDT", kline.OneHour, time.Now().Add(-time.Hour*24*500), time.Now(), 500)
	if err != nil {
		t.Error("Okx GetCandlesticks() error", err)
	}
}

func TestGetCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCandlesticksHistory(context.Background(), "BTC-USDT", kline.OneHour, time.Unix(time.Now().Unix()-int64(time.Minute), 3), time.Now(), 3)
	if err != nil {
		t.Error("Okx GetCandlesticksHistory() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTrades(context.Background(), "BTC-USDT", 3)
	if err != nil {
		t.Error("Okx GetTrades() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetTradesHistory(context.Background(), "BTC-USDT", "", "", 2); err != nil {
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

func TestGetBlockTickers(t *testing.T) {
	t.Parallel()
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

func TestGetBlockTrade(t *testing.T) {
	t.Parallel()
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

func TestGetDeliveryHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetDeliveryHistory(context.Background(), "FUTURES", "BTC-USDT", time.Time{}, time.Time{}, 3)
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

func TestGetSingleFundingRate(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetSingleFundingRate(context.Background(), "BTC-USD-SWAP"); err != nil {
		t.Error("okx GetSingleFundingRate() error", err)
	}
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetFundingRateHistory(context.Background(), "BTC-USD-SWAP", time.Time{}, time.Time{}, 2); err != nil {
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

func TestGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	r, err := ok.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ok.GetEstimatedDeliveryPrice(context.Background(), r[0].String()); err != nil {
		t.Error("Okx GetEstimatedDeliveryPrice() error", err)
	}
}

func TestGetDiscountRateAndInterestFreeQuota(t *testing.T) {
	t.Parallel()
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
	insts, err := ok.FetchTradablePairs(context.Background(), asset.Margin)
	if err != nil {
		t.Skip(err)
	}
	if _, err := ok.GetLiquidationOrders(context.Background(), &LiquidationOrderRequestParams{
		InstrumentType: okxInstTypeMargin,
		Underlying:     insts[0].String(),
		Currency:       currency.BTC,
		Limit:          2,
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

func TestGetInsuranceFundInformation(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetInsuranceFundInformation(context.Background(), &InsuranceFundInformationRequestParams{
		InstrumentType: "FUTURES",
		Underlying:     "BTC-USDT",
		Limit:          2,
	}); err != nil {
		t.Error("Okx GetInsuranceFundInformation() error", err)
	}
}

func TestCurrencyUnitConvert(t *testing.T) {
	t.Parallel()
	if _, err := ok.CurrencyUnitConvert(context.Background(), "BTC-USD-SWAP", 1, 3500, 1, ""); err != nil {
		t.Error("Okx CurrencyUnitConvert() error", err)
	}
}

// Trading related endpoints test functions.
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
	if _, err := ok.GetMarginLendingRatio(context.Background(), "BTC", time.Time{}, time.Time{}, kline.FiveMin); err != nil {
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

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.PlaceOrder(context.Background(), &PlaceOrderRequestParam{
		InstrumentID: "BTC-USDC",
		TradeMode:    "cross",
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       2.6,
		Price:        2.1,
		Currency:     "BTC",
	}, asset.Margin); err != nil {
		t.Error("Okx PlaceOrder() error", err)
	}
}

const placeMultipleOrderParamsJSON = `[{"instId":"BTC-USDT","tdMode":"cash","clOrdId":"b159","side":"buy","ordType":"limit","px":"2.15","sz":"2"},{"instId":"BTC-USDT","tdMode":"cash","clOrdId":"b15","side":"buy","ordType":"limit","px":"2.15","sz":"2"}]`

func TestPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var params []PlaceOrderRequestParam
	err := json.Unmarshal([]byte(placeMultipleOrderParamsJSON), &params)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = ok.PlaceMultipleOrders(context.Background(),
		params); err != nil {
		t.Error("Okx PlaceMultipleOrders() error", err)
	}
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelSingleOrder(context.Background(),
		CancelOrderRequestParam{
			InstrumentID: "BTC-USDT",
			OrderID:      "2510789768709120",
		}); err != nil {
		t.Error("Okx CancelOrder() error", err)
	}
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelMultipleOrders(context.Background(), []CancelOrderRequestParam{{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
	}}); err != nil {
		t.Error("Okx CancelMultipleOrders() error", err)
	}
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.AmendOrder(context.Background(), &AmendOrderRequestParams{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}); err != nil {
		t.Error("Okx AmendOrder() error", err)
	}
}
func TestAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.AmendMultipleOrders(context.Background(), []AmendOrderRequestParams{{
		InstrumentID: "BTC-USDT",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}}); err != nil {
		t.Error("Okx AmendMultipleOrders() error", err)
	}
}

func TestClosePositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.ClosePositions(context.Background(), &ClosePositionsRequestParams{
		InstrumentID: "BTC-USDT",
		MarginMode:   "cross",
		Currency:     "BTC",
	}); err != nil {
		t.Error("Okc ClosePositions() error", err)
	}
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetOrderDetail(context.Background(), &OrderDetailRequestParam{
		InstrumentID: "BTC-USDT",
		OrderID:      "2510789768709120",
	}); !strings.Contains(err.Error(), "Order does not exist") {
		t.Error("Okx GetOrderDetail() error", err)
	}
}

func TestGetOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetOrderList(context.Background(), &OrderListRequestParams{
		Limit: 1,
	}); err != nil {
		t.Error("Okx GetOrderList() error", err)
	}
}

func TestGet7And3MonthDayOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

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

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetTransactionDetailsLast3Days(context.Background(), &TransactionDetailRequestParams{
		InstrumentType: "MARGIN",
		Limit:          1,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.PlaceStopOrder(context.Background(), &AlgoOrderParams{
		TakeProfitTriggerPriceType: "index",
		InstrumentID:               "BTC-USDT",
		OrderType:                  "conditional",
		Side:                       order.Sell,
		TradeMode:                  "isolated",
		Size:                       12,

		TakeProfitTriggerPrice: 12335,
		TakeProfitOrderPrice:   1234,
	}); err != nil {
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
	}); err != nil {
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
	}); err != nil {
		t.Error("Okx PlaceIceburgOrder() error", err)
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
	}); err != nil {
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
	}); err != nil {
		t.Error("Okx TriggerAlogOrder() error", err)
	}
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelAlgoOrder(context.Background(), []AlgoOrderCancelParams{
		{
			InstrumentID: "BTC-USDT",
			AlgoOrderID:  "90994943",
		},
	}); err != nil {
		t.Error("Okx CancelAlgoOrder() error", err)
	}
}

func TestCancelAdvanceAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelAdvanceAlgoOrder(context.Background(), []AlgoOrderCancelParams{{
		InstrumentID: "BTC-USDT",
		AlgoOrderID:  "90994943",
	}}); err != nil {
		t.Error("Okx CancelAdvanceAlgoOrder() error", err)
	}
}

func TestGetAlgoOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetAlgoOrderList(context.Background(), "conditional", "", "", "", "", time.Time{}, time.Time{}, 1); err != nil {
		t.Error("Okx GetAlgoOrderList() error", err)
	}
}

func TestGetAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetAlgoOrderHistory(context.Background(), "conditional", "effective", "", "", "", time.Time{}, time.Time{}, 1); err != nil {
		t.Error("Okx GetAlgoOrderList() error", err)
	}
}

func TestGetEasyConvertCurrencyList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetEasyConvertCurrencyList(context.Background()); err != nil {
		t.Errorf("%s GetEasyConvertCurrencyList() error %v", ok.Name, err)
	}
}

func TestGetOneClickRepayCurrencyList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetOneClickRepayCurrencyList(context.Background(), "cross"); err != nil && !strings.Contains(err.Error(), "Parameter acctLv  error") {
		t.Error(err)
	}
}

func TestPlaceEasyConvert(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.PlaceEasyConvert(context.Background(),
		PlaceEasyConvertParam{
			FromCurrency: []string{"BTC"},
			ToCurrency:   "USDT"}); err != nil {
		t.Errorf("%s PlaceEasyConvert() error %v", ok.Name, err)
	}
}

func TestGetEasyConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetEasyConvertHistory(context.Background(), time.Time{}, time.Time{}, 1); err != nil {
		t.Errorf("%s GetEasyConvertHistory() error %v", ok.Name, err)
	}
}

func TestGetOneClickRepayHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetOneClickRepayHistory(context.Background(), time.Time{}, time.Time{}, 1); err != nil && !strings.Contains(err.Error(), "Parameter acctLv  error") {
		t.Errorf("%s GetOneClickRepayHistory() error %v", ok.Name, err)
	}
}

func TestTradeOneClickRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.TradeOneClickRepay(context.Background(), TradeOneClickRepayParam{
		DebtCurrency:  []string{"BTC"},
		RepayCurrency: "USDT",
	}); err != nil {
		t.Errorf("%s TradeOneClickRepay() error %v", ok.Name, err)
	}
}

func TestGetCounterparties(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetCounterparties(context.Background()); err != nil && !strings.Contains(err.Error(), "code: 70006 message: Does not meet the minimum asset requirement.") {
		t.Error("Okx GetCounterparties() error", err)
	}
}

const createRFQInputJSON = `{"anonymous": true,"counterparties":["Trader1","Trader2"],"clRfqId":"rfq01","legs":[{"sz":"25","side":"buy","instId":"BTCUSD-221208-100000-C"},{"sz":"150","side":"buy","instId":"ETH-USDT","tgtCcy":"base_ccy"}]}`

func TestCreateRFQ(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var input CreateRFQInput
	if err := json.Unmarshal([]byte(createRFQInputJSON), &input); err != nil {
		t.Error("Okx Decerializing to CreateRFQInput", err)
	}
	if _, err := ok.CreateRFQ(context.Background(), input); err != nil {
		t.Error("Okx CreateRFQ() error", err)
	}
}

func TestCancelRFQ(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelRFQ(context.Background(), CancelRFQRequestParam{})
	if err != nil && !errors.Is(err, errMissingRFQIDANDClientSuppliedRFQID) {
		t.Errorf("Okx CancelRFQ() expecting %v, but found %v", errMissingRFQIDANDClientSuppliedRFQID, err)
	}
	_, err = ok.CancelRFQ(context.Background(), CancelRFQRequestParam{
		ClientSuppliedRFQID: "somersdjskfjsdkfjxvxv",
	})
	if err != nil {
		t.Error("Okx CancelRFQ() error", err)
	}
}

func TestMultipleCancelRFQ(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelMultipleRFQs(context.Background(), CancelRFQRequestsParam{})
	if err != nil && !errors.Is(err, errMissingRFQIDANDClientSuppliedRFQID) {
		t.Errorf("Okx CancelMultipleRFQs() expecting %v, but found %v", errMissingRFQIDANDClientSuppliedRFQID, err)
	}
	_, err = ok.CancelMultipleRFQs(context.Background(), CancelRFQRequestsParam{
		ClientSuppliedRFQID: []string{"somersdjskfjsdkfjxvxv"},
	})
	if err != nil {
		t.Error("Okx CancelMultipleRFQs() error", err)
	}
}

func TestCancelAllRFQs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelAllRFQs(context.Background()); err != nil {
		t.Errorf("%s CancelAllRFQs() error %v", ok.Name, err)
	}
}

func TestExecuteQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.ExecuteQuote(context.Background(), ExecuteQuoteParams{})
	if err != nil && !errors.Is(err, errMissingRfqIDOrQuoteID) {
		t.Errorf("Okx ExecuteQuote() expected %v, but found %v", errMissingRfqIDOrQuoteID, err)
	}
	if _, err = ok.ExecuteQuote(context.Background(), ExecuteQuoteParams{
		RfqID:   "22540",
		QuoteID: "84073",
	}); err != nil {
		t.Error("Okx ExecuteQuote() error", err)
	}
}

func TestSetQuoteProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

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
		}}); err != nil {
		t.Errorf("%s SetQuoteProducts() error %v", ok.Name, err)
	}
}

func TestResetMMPStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.ResetMMPStatus(context.Background()); err != nil && !strings.Contains(err.Error(), "No permission to use this API") {
		t.Errorf("%s ResetMMPStatus() error %v", ok.Name, err)
	}
}

func TestCreateQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

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
	}); err != nil {
		t.Errorf("%s CreateQuote() error %v", ok.Name, err)
	}
}

func TestCancelQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelQuote(context.Background(), CancelQuoteRequestParams{}); err != nil && !errors.Is(err, errMissingQuoteIDOrClientSuppliedQuoteID) {
		t.Error("Okx CancelQuote() error", err)
	}
	if _, err := ok.CancelQuote(context.Background(), CancelQuoteRequestParams{
		QuoteID: "1234",
	}); err != nil {
		t.Error("Okx CancelQuote() error", err)
	}
	if _, err := ok.CancelQuote(context.Background(), CancelQuoteRequestParams{
		ClientSuppliedQuoteID: "1234",
	}); err != nil {
		t.Error("Okx CancelQuote() error", err)
	}
}

func TestCancelMultipleQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelMultipleQuote(context.Background(), CancelQuotesRequestParams{}); err != nil && !errors.Is(errMissingEitherQuoteIDAOrClientSuppliedQuoteIDs, err) {
		t.Error("Okx CancelQuote() error", err)
	}
	if _, err := ok.CancelMultipleQuote(context.Background(), CancelQuotesRequestParams{
		QuoteIDs: []string{"1150", "1151", "1152"},
		// Block trades require a minimum of $100,000 in assets in your trading account
	}); err != nil {
		t.Error("Okx CancelQuote() error", err)
	}
}

func TestCancelAllQuotes(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	time, err := ok.CancelAllQuotes(context.Background())
	switch {
	case err != nil:
		t.Error("Okx CancelAllQuotes() error", err)
	case err == nil && time.IsZero():
		t.Error("Okx CancelAllQuotes() zero timestamp message ")
	}
}

func TestGetRFQs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetRfqs(context.Background(), &RfqRequestParams{
		Limit: 1,
	}); err != nil {
		t.Error("Okx GetRfqs() error", err)
	}
}

func TestGetQuotes(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetQuotes(context.Background(), &QuoteRequestParams{
		Limit: 3,
	}); err != nil {
		t.Error("Okx GetQuotes() error", err)
	}
}

func TestGetRFQTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetRFQTrades(context.Background(), &RFQTradesRequestParams{
		Limit: 1,
	}); err != nil {
		t.Error("Okx GetRFQTrades() error", err)
	}
}

func TestGetPublicTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetPublicTrades(context.Background(), "", "", 3); err != nil {
		t.Error("Okx GetPublicTrades() error", err)
	}
}

func TestGetFundingCurrencies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetFundingCurrencies(context.Background()); err != nil {
		t.Error("Okx  GetFundingCurrencies() error", err)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetBalance(context.Background(), ""); err != nil {
		t.Error("Okx GetBalance() error", err)
	}
}

func TestGetAccountAssetValuation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetAccountAssetValuation(context.Background(), ""); err != nil {
		t.Error("Okx  GetAccountAssetValuation() error", err)
	}
}

func TestFundingTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.FundingTransfer(context.Background(), &FundingTransferRequestInput{
		Amount:   12.000,
		To:       "6",
		From:     "18",
		Currency: "BTC",
	}); err != nil {
		t.Error("Okx FundingTransfer() error", err)
	}
}

func TestGetFundsTransferState(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetFundsTransferState(context.Background(), "754147", "1232", 1); err != nil && !strings.Contains(err.Error(), "Parameter transId  error") {
		t.Error("Okx GetFundsTransferState() error", err)
	}
}

func TestGetAssetBillsDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetAssetBillsDetails(context.Background(), "", "", time.Time{}, time.Time{}, 0, 1)
	if err != nil {
		t.Error("Okx GetAssetBillsDetail() error", err)
	}
}

func TestGetLightningDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetLightningDeposits(context.Background(), "BTC", 1.00, 0); err != nil && !strings.Contains(err.Error(), "58355") {
		t.Error("Okx GetLightningDeposits() error", err)
	}
}

func TestGetCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetCurrencyDepositAddress(context.Background(), "BTC"); err != nil {
		t.Error("Okx GetCurrencyDepositAddress() error", err)
	}
}

func TestGetCurrencyDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetCurrencyDepositHistory(context.Background(), "BTC", "", "", time.Time{}, time.Time{}, 0, 1); err != nil {
		t.Error("Okx GetCurrencyDepositHistory() error", err)
	}
}

func TestWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.Withdrawal(context.Background(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: "BTC", WithdrawalDestination: "4", ToAddress: core.BitcoinDonationAddress})
	if err != nil {
		t.Error("Okx Withdrawal error", err)
	}
}

func TestLightningWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.LightningWithdrawal(context.Background(), LightningWithdrawalRequestInput{
		Currency: currency.BTC.String(),
		Invoice:  "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hz",
	}); err != nil {
		t.Error("Okx LightningWithdrawal() error", err)
	}
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelWithdrawal(context.Background(), "fjasdfkjasdk"); err != nil {
		t.Error("Okx CancelWithdrawal() error", err.Error())
	}
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetWithdrawalHistory(context.Background(), "BTC", "", "", "", "", time.Time{}, time.Time{}, 1); err != nil {
		t.Error("Okx GetWithdrawalHistory() error", err)
	}
}

func TestSmallAssetsConvert(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.SmallAssetsConvert(context.Background(), []string{"BTC", "USDT"}); err != nil {
		t.Error("Okx SmallAssetsConvert() error", err)
	}
}

func TestGetSavingBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetSavingBalance(context.Background(), "BTC"); err != nil {
		t.Error("Okx GetSavingBalance() error", err)
	}
}

func TestSavingsPurchase(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.SavingsPurchaseOrRedemption(context.Background(), &SavingsPurchaseRedemptionInput{
		Amount:     123.4,
		Currency:   "BTC",
		Rate:       1,
		ActionType: "purchase",
	}); err != nil {
		t.Error("Okx SavingsPurchaseOrRedemption() error", err)
	}
	if _, err := ok.SavingsPurchaseOrRedemption(context.Background(), &SavingsPurchaseRedemptionInput{
		Amount:     123.4,
		Currency:   "BTC",
		Rate:       1,
		ActionType: "redempt",
	}); err != nil {
		t.Error("Okx SavingsPurchaseOrRedemption() error", err)
	}
}

func TestSetLendingRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.SetLendingRate(context.Background(), LendingRate{Currency: "BTC", Rate: 2}); err != nil {
		t.Error("Okx SetLendingRate() error", err)
	}
}

func TestGetLendingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetLendingHistory(context.Background(), "USDT", time.Time{}, time.Time{}, 1); err != nil {
		t.Error("Okx GetLendingHostory() error", err)
	}
}

func TestGetPublicBorrowInfo(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetPublicBorrowInfo(context.Background(), ""); err != nil {
		t.Error("Okx GetPublicBorrowInfo() error", err)
	}
	if _, err := ok.GetPublicBorrowInfo(context.Background(), "USDT"); err != nil {
		t.Error("Okx GetPublicBorrowInfo() error", err)
	}
}

func TestGetPublicBorrowHistory(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetPublicBorrowHistory(context.Background(), "USDT", time.Time{}, time.Time{}, 1); err != nil {
		t.Error("Okx GetPublicBorrowHistory() error", err)
	}
}

func TestGetConvertCurrencies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetConvertCurrencies(context.Background()); err != nil {
		t.Error("Okx GetConvertCurrencies() error", err)
	}
}

func TestGetConvertCurrencyPair(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetConvertCurrencyPair(context.Background(), "USDT", "BTC"); err != nil {
		t.Error("Okx GetConvertCurrencyPair() error", err)
	}
}

func TestEstimateQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

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

func TestConvertTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.ConvertTrade(context.Background(), &ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          "Buy",
		Size:          2,
		SizeCurrency:  "USDT",
		QuoteID:       "quoterETH-USDT16461885104612381",
	}); err != nil {
		t.Error("Okx ConvertTrade() error", err)
	}
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetConvertHistory(context.Background(), time.Time{}, time.Time{}, 1, ""); err != nil {
		t.Error("Okx GetConvertHistory() error", err)
	}
}

func TestGetNonZeroAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetNonZeroBalances(context.Background(), ""); err != nil {
		t.Error("Okx GetBalance() error", err)
	}
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetPositions(context.Background(), "", "", ""); err != nil {
		t.Error("Okx GetPositions() error", err)
	}
}

func TestGetPositionsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetPositionsHistory(context.Background(), "", "", "", 0, 1, time.Time{}, time.Time{}); err != nil {
		t.Error("Okx GetPositionsHistory() error", err)
	}
}

func TestGetAccountAndPositionRisk(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetAccountAndPositionRisk(context.Background(), ""); err != nil {
		t.Error("Okx GetAccountAndPositionRisk() error", err)
	}
}

func TestGetBillsDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetBillsDetailLast7Days(context.Background(), &BillsDetailQueryParameter{
		Limit: 3,
	}); err != nil {
		t.Error("Okx GetBillsDetailLast7Days() error", err)
	}
}

func TestGetAccountConfiguration(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetAccountConfiguration(context.Background()); err != nil {
		t.Error("Okx GetAccountConfiguration() error", err)
	}
}

func TestSetPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.SetPositionMode(context.Background(), "net_mode"); err != nil {
		t.Error("Okx SetPositionMode() error", err)
	}
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.SetLeverage(context.Background(), SetLeverageInput{
		Currency:     "USDT",
		Leverage:     5,
		MarginMode:   "cross",
		InstrumentID: "BTC-USDT",
	}); err != nil && !errors.Is(err, errNoValidResponseFromServer) {
		t.Error("Okx SetLeverage() error", err)
	}
}

func TestGetMaximumBuySellAmountOROpenAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetMaximumBuySellAmountOROpenAmount(context.Background(), "BTC-USDT", "cross", "BTC", "", 5); err != nil {
		t.Error("Okx GetMaximumBuySellAmountOROpenAmount() error", err)
	}
}

func TestGetMaximumAvailableTradableAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetMaximumAvailableTradableAmount(context.Background(), "BTC-USDT", "BTC", "cross", true, 123); err != nil && !strings.Contains(err.Error(), "51010") {
		t.Error("Okx GetMaximumAvailableTradableAmount() error", err)
	}
}

func TestIncreaseDecreaseMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.IncreaseDecreaseMargin(context.Background(), IncreaseDecreaseMarginInput{
		InstrumentID: "BTC-USDT",
		PositionSide: "long",
		Type:         "add",
		Amount:       1000,
		Currency:     "USD",
	}); err != nil {
		t.Error("Okx IncreaseDecreaseMargin() error", err)
	}
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetLeverage(context.Background(), "BTC-USDT", "cross"); err != nil {
		t.Error("Okx GetLeverage() error", err)
	}
}

func TestGetMaximumLoanOfInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetMaximumLoanOfInstrument(context.Background(), "ZRX-BTC", "isolated", "ZRX"); err != nil && !strings.Contains(err.Error(), "51010") {
		t.Error("Okx GetMaximumLoanOfInstrument() error", err)
	}
}

func TestGetTradeFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetTradeFee(context.Background(), "SPOT", "", ""); err != nil {
		t.Error("Okx GetTradeFeeRate() error", err)
	}
}

func TestGetInterestAccruedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetInterestAccruedData(context.Background(), 0, 1, "", "", "", time.Time{}, time.Time{}); err != nil {
		t.Error("Okx GetInterestAccruedData() error", err)
	}
}

func TestGetInterestRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetInterestRate(context.Background(), ""); err != nil {
		t.Error("Okx GetInterestRate() error", err)
	}
}

func TestSetGreeks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.SetGreeks(context.Background(), "PA"); err != nil {
		t.Error("Okx SetGreeks() error", err)
	}
}

func TestIsolatedMarginTradingSettings(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.IsolatedMarginTradingSettings(context.Background(), IsolatedMode{
		IsoMode:        "autonomy",
		InstrumentType: "MARGIN",
	}); err != nil {
		t.Error("Okx IsolatedMarginTradingSettings() error", err)
	}
}

func TestGetMaximumWithdrawals(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetMaximumWithdrawals(context.Background(), "BTC"); err != nil {
		t.Error("Okx GetMaximumWithdrawals() error", err)
	}
}

func TestGetAccountRiskState(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetAccountRiskState(context.Background()); err != nil && !strings.Contains(err.Error(), "51010") {
		t.Error("Okx GetAccountRiskState() error", err)
	}
}

func TestVIPLoansBorrowAndRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.VIPLoansBorrowAndRepay(context.Background(), LoanBorrowAndReplayInput{Currency: "BTC", Side: "borrow", Amount: 12}); err != nil &&
		!strings.Contains(err.Error(), "Your account does not support VIP loan") {
		t.Error("Okx VIPLoansBorrowAndRepay() error", err)
	}
}

func TestGetBorrowAndRepayHistoryForVIPLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetBorrowAndRepayHistoryForVIPLoans(context.Background(), "", time.Time{}, time.Time{}, 3); err != nil {
		t.Error("Okx GetBorrowAndRepayHistoryForVIPLoans() error", err)
	}
}

func TestGetBorrowInterestAndLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetBorrowInterestAndLimit(context.Background(), 1, "BTC"); err != nil && !strings.Contains(err.Error(), "59307") { // You are not eligible for VIP loans
		t.Error("Okx GetBorrowInterestAndLimit() error", err)
	}
}

func TestPositionBuilder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.PositionBuilder(context.Background(), PositionBuilderInput{
		ImportExistingPosition: true,
	}); err != nil {
		t.Error("Okx PositionBuilder() error", err)
	}
}

func TestGetGreeks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetGreeks(context.Background(), ""); err != nil && !strings.Contains(err.Error(), "Unsupported operation") {
		t.Error("Okx GetGreeks() error", err)
	}
}

func TestGetPMLimitation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetPMLimitation(context.Background(), "SWAP", "BTC-USDT"); err != nil {
		t.Errorf("%s GetPMLimitation() error %v", ok.Name, err)
	}
}

func TestViewSubaccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.ViewSubAccountList(context.Background(), false, "", time.Time{}, time.Time{}, 2); err != nil {
		t.Error("Okx ViewSubaccountList() error", err)
	}
}

func TestResetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.ResetSubAccountAPIKey(context.Background(), &SubAccountAPIKeyParam{
		SubAccountName:   "sam",
		APIKey:           apiKey,
		APIKeyPermission: "trade",
	}); err != nil && !strings.Contains(err.Error(), "Parameter subAcct can not be empty.") {
		t.Errorf("%s ResetSubAccountAPIKey() error %v", ok.Name, err)
	}
	if _, err := ok.ResetSubAccountAPIKey(context.Background(), &SubAccountAPIKeyParam{
		SubAccountName: "sam",
		APIKey:         apiKey,
		Permissions:    []string{"trade", "read"},
	}); err != nil && !strings.Contains(err.Error(), "Parameter subAcct can not be empty.") {
		t.Errorf("%s ResetSubAccountAPIKey() error %v", ok.Name, err)
	}
}

func TestGetSubaccountTradingBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetSubaccountTradingBalance(context.Background(), ""); err != nil && !errors.Is(err, errMissingRequiredParameterSubaccountName) {
		t.Errorf("Okx GetSubaccountTradingBalance() expecting \"%v\", but found \"%v\"", errMissingRequiredParameterSubaccountName, err)
	}
	if _, err := ok.GetSubaccountTradingBalance(context.Background(), "test1"); err != nil && !strings.Contains(err.Error(), "sub-account does not exist") {
		t.Error("Okx GetSubaccountTradingBalance() error", err)
	}
}

func TestGetSubaccountFundingBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetSubaccountFundingBalance(context.Background(), "test1", ""); err != nil && !strings.Contains(err.Error(), "Sub-account test1 does not exists") && !strings.Contains(err.Error(), "59510") {
		t.Error("Okx GetSubaccountFundingBalance() error", err)
	}
}

func TestHistoryOfSubaccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.HistoryOfSubaccountTransfer(context.Background(), "", "0", "", time.Time{}, time.Time{}, 1); err != nil {
		t.Error("Okx HistoryOfSubaccountTransfer() error", err)
	}
}

func TestMasterAccountsManageTransfersBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(context.Background(), SubAccountAssetTransferParams{Currency: "BTC", Amount: 1200, From: 9, To: 9, FromSubAccount: "", ToSubAccount: "", LoanTransfer: true}); err != nil && !errors.Is(err, errInvalidSubaccount) {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", err)
	}
	if _, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(context.Background(), SubAccountAssetTransferParams{Currency: "BTC", Amount: 1200, From: 8, To: 8, FromSubAccount: "", ToSubAccount: "", LoanTransfer: true}); err != nil && !errors.Is(err, errInvalidSubaccount) {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", err)
	}
	if _, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(context.Background(), SubAccountAssetTransferParams{Currency: "BTC", Amount: 1200, From: 6, To: 6, FromSubAccount: "test1", ToSubAccount: "test2", LoanTransfer: true}); err != nil && !strings.Contains(err.Error(), "Sub-account test1 does not exists") {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", err)
	}
}

func TestSetPermissionOfTransferOut(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.SetPermissionOfTransferOut(context.Background(), PermissionOfTransfer{SubAcct: "Test1"}); err != nil && !strings.Contains(err.Error(), "Sub-account does not exist") {
		t.Error("Okx SetPermissionOfTransferOut() error", err)
	}
}

func TestGetCustodyTradingSubaccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetCustodyTradingSubaccountList(context.Background(), ""); err != nil {
		t.Error("Okx GetCustodyTradingSubaccountList() error", err)
	}
}

const gridTradingPlaceOrder = `{"instId": "BTC-USD-SWAP","algoOrdType": "contract_grid","maxPx": "5000","minPx": "400","gridNum": "10","runType": "1","sz": "200", "direction": "long","lever": "2"}`

func TestPlaceGridAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var input GridAlgoOrder
	if err := json.Unmarshal([]byte(gridTradingPlaceOrder), &input); err != nil {
		t.Error("Okx Decerializing to GridALgoOrder error", err)
	}
	if _, err := ok.PlaceGridAlgoOrder(context.Background(), &input); err != nil {
		t.Error("Okx PlaceGridAlgoOrder() error", err)
	}
}

const gridOrderAmendAlgo = `{
    "algoId":"448965992920907776",
    "instId":"BTC-USDT",
    "slTriggerPx":"1200",
    "tpTriggerPx":""
}`

func TestAmendGridAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var input GridAlgoOrderAmend
	if err := json.Unmarshal([]byte(gridOrderAmendAlgo), &input); err != nil {
		t.Error("Okx Decerializing to GridAlgoOrderAmend error", err)
	}
	if _, err := ok.AmendGridAlgoOrder(context.Background(), input); err != nil {
		t.Error("Okx AmendGridAlgoOrder() error", err)
	}
}

const stopGridAlgoOrderJSON = `{"algoId":"198273485",	"instId":"BTC-USDT",	"stopType":"1",	"algoOrdType":"grid"}`

func TestStopGridAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var resp StopGridAlgoOrderRequest
	if err := json.Unmarshal([]byte(stopGridAlgoOrderJSON), &resp); err != nil {
		t.Error("error deserializing to StopGridAlgoOrder error", err)
	}
	if _, err := ok.StopGridAlgoOrder(context.Background(), []StopGridAlgoOrderRequest{
		resp,
	}); err != nil && !strings.Contains(err.Error(), "The strategy does not exist or has stopped") {
		t.Error("Okx StopGridAlgoOrder() error", err)
	}
}

func TestGetGridAlgoOrdersList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetGridAlgoOrdersList(context.Background(), "grid", "", "", "", "", "", 1); err != nil {
		t.Error("Okx GetGridAlgoOrdersList() error", err)
	}
}

func TestGetGridAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetGridAlgoOrderHistory(context.Background(), "contract_grid", "", "", "", "", "", 1); err != nil {
		t.Error("Okx GetGridAlgoOrderHistory() error", err)
	}
}

func TestGetGridAlgoOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetGridAlgoOrderDetails(context.Background(), "grid", ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderDetails() expecting %v, but found %v error", errMissingAlgoOrderID, err)
	}
	if _, err := ok.GetGridAlgoOrderDetails(context.Background(), "grid", "7878"); err != nil && !strings.Contains(err.Error(), "Order does not exist") {
		t.Error("Okx GetGridAlgoOrderDetails() error", err)
	}
}

func TestGetGridAlgoSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetGridAlgoSubOrders(context.Background(), "", "", "", "", "", "", 2); err != nil && !errors.Is(err, errMissingAlgoOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingAlgoOrderType)
	}
	if _, err := ok.GetGridAlgoSubOrders(context.Background(), "grid", "", "", "", "", "", 2); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingAlgoOrderID)
	}
	if _, err := ok.GetGridAlgoSubOrders(context.Background(), "grid", "1234", "", "", "", "", 2); err != nil && !errors.Is(err, errMissingSubOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingSubOrderType)
	}
	if _, err := ok.GetGridAlgoSubOrders(context.Background(), "grid", "1234", "live", "", "", "", 2); err != nil && !errors.Is(err, errMissingSubOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingSubOrderType)
	}
}

const spotGridAlgoOrderPosition = `{"adl": "1","algoId": "449327675342323712","avgPx": "29215.0142857142857149","cTime": "1653400065917","ccy": "USDT","imr": "2045.386","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29206.7","lever": "5","liqPx": "661.1684795867162","markPx": "29213.9","mgnMode": "cross","mgnRatio": "217.19370606167573","mmr": "40.907720000000005","notionalUsd": "10216.70307","pos": "35","posSide": "net","uTime": "1653400066938","upl": "1.674999999999818","uplRatio": "0.0008190504784478"}`

func TestGetGridAlgoOrderPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	var resp AlgoOrderPosition
	if err := json.Unmarshal([]byte(spotGridAlgoOrderPosition), &resp); err != nil {
		t.Error("Okx Decerializing to AlgoOrderPosition error", err)
	}
	if _, err := ok.GetGridAlgoOrderPositions(context.Background(), "", ""); err != nil && !errors.Is(err, errInvalidAlgoOrderType) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errInvalidAlgoOrderType, err)
	}
	if _, err := ok.GetGridAlgoOrderPositions(context.Background(), "contract_grid", ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
	if _, err := ok.GetGridAlgoOrderPositions(context.Background(), "contract_grid", ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
	if _, err := ok.GetGridAlgoOrderPositions(context.Background(), "contract_grid", "448965992920907776"); err != nil && !strings.Contains(err.Error(), "The strategy does not exist or has stopped") {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
}

func TestSpotGridWithdrawProfit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.SpotGridWithdrawProfit(context.Background(), ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx SpotGridWithdrawProfit() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
	if _, err := ok.SpotGridWithdrawProfit(context.Background(), "1234"); err != nil {
		t.Error("Okx SpotGridWithdrawProfit() error", err)
	}
}

func TestComputeMarginBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.AdjustMarginBalance(context.Background(), MarginBalanceParam{
		AlgoID: "1234",
		Type:   "add",
		Amount: 12345,
	}); err != nil {
		t.Errorf("%s AdjustMarginBalance() error %v", ok.Name, err)
	}
}

const gridAIParamJSON = `{"algoOrdType": "grid","annualizedRate": "1.5849","ccy": "USDT","direction": "",	"duration": "7D","gridNum": "5","instId": "BTC-USDT","lever": "0","maxPx": "21373.3","minInvestment": "0.89557758",	"minPx": "15544.2",	"perMaxProfitRate": "0.0733865364573281","perMinProfitRate": "0.0561101403446263","runType": "1"}`

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
func TestGetOffers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetOffers(context.Background(), "", "", ""); err != nil {
		t.Errorf("%s GetOffers() error %v", ok.Name, err)
	}
}

func TestPurchase(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.Purchase(context.Background(), PurchaseRequestParam{
		ProductID: "1234",
		InvestData: []PurchaseInvestDataItem{
			{
				Currency: "BTC",
				Amount:   100,
			},
			{
				Currency: "ETH",
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelPurchaseOrRedemption(context.Background(), CancelFundingParam{
		OrderID:      "754147",
		ProtocolType: "defi",
	}); err != nil && !strings.Contains(err.Error(), "Order not found") {
		t.Errorf("%s CancelPurchaseOrRedemption() error %v", ok.Name, err)
	}
}

func TestGetEarnActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetEarnActiveOrders(context.Background(), "", "", "", ""); err != nil {
		t.Errorf("%s GetEarnActiveOrders() error %v", ok.Name, err)
	}
}

func TestGetFundingOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetFundingOrderHistory(context.Background(), "", "", "", time.Time{}, time.Time{}, 1); err != nil {
		t.Errorf("%s GetFundingOrderHistory() error %v", ok.Name, err)
	}
}

func TestSystemStatusResponse(t *testing.T) {
	t.Parallel()
	if _, err := ok.SystemStatusResponse(context.Background(), "completed"); err != nil {
		t.Error("Okx SystemStatusResponse() error", err)
	}
}

/**********************************  Wrapper Functions **************************************/

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	if _, err := ok.FetchTradablePairs(context.Background(), asset.Options); err != nil {
		t.Error("Okx FetchTradablePairs() error", err)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	if err := ok.UpdateTradablePairs(context.Background(), true); err != nil {
		t.Error("Okx UpdateTradablePairs() error", err)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()

	type limitTest struct {
		pair currency.Pair
		step float64
		min  float64
	}

	tests := map[asset.Item][]limitTest{
		asset.Spot: {
			{currency.NewPair(currency.ETH, currency.USDT), 0.01, 0.0001},
			{currency.NewPair(currency.BTC, currency.USDT), 0.1, 0.00001},
		},
		asset.Margin: {
			{currency.NewPair(currency.ETH, currency.USDT), 0.01, 0.0001},
			{currency.NewPair(currency.ETH, currency.BTC), 0.00001, 0.0001},
		},
	}

	for _, a := range []asset.Item{asset.PerpetualSwap, asset.Futures, asset.Options} {
		pairs, err := ok.FetchTradablePairs(context.Background(), a)
		if err != nil {
			t.Errorf("Error fetching dated %s pairs for test: %v", a, err)
		}
		stepIncr := 0.1
		if a == asset.Options {
			stepIncr = 0.0005
		}

		tests[a] = []limitTest{{pairs[0], stepIncr, 1}}
	}

	for _, a := range ok.GetAssetTypes(false) {
		if err := ok.UpdateOrderExecutionLimits(context.Background(), a); err != nil {
			t.Error("Okx UpdateOrderExecutionLimits() error", err)
			continue
		}

		for _, tt := range tests[a] {
			limits, err := ok.GetOrderExecutionLimits(a, tt.pair)
			if err != nil {
				t.Errorf("Okx GetOrderExecutionLimits() error during TestUpdateOrderExecutionLimits; Asset: %s Pair: %s Err: %v", a, tt.pair, err)
				continue
			}

			if got := limits.PriceStepIncrementSize; got != tt.step {
				t.Errorf("Okx UpdateOrderExecutionLimits wrong PriceStepIncrementSize; Asset: %s Pair: %s Expected: %v Got: %v", a, tt.pair, tt.step, got)
			}

			if got := limits.MinimumBaseAmount; got != tt.min {
				t.Errorf("Okx UpdateOrderExecutionLimits wrong MinAmount; Pair: %s Expected: %v Got: %v", tt.pair, tt.min, got)
			}
		}
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	if _, err := ok.UpdateTicker(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot); err != nil {
		t.Error("Okx UpdateTicker() error", err)
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	if err := ok.UpdateTickers(context.Background(), asset.Spot); err != nil {
		t.Error("Okx UpdateTicker() error", err)
	}
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.FetchTicker(context.Background(), currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")), asset.PerpetualSwap)
	if err != nil {
		t.Error("Okx FetchTicker() error", err)
	}
	if _, err = ok.FetchTicker(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot); err != nil {
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
	if _, err := ok.UpdateOrderbook(context.Background(), currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")), asset.Spot); err != nil {
		t.Error("Okx UpdateOrderbook() error", err)
	}
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.UpdateAccountInfo(context.Background(), asset.Spot); err != nil {
		t.Error("Okx UpdateAccountInfo() error", err)
	}
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.FetchAccountInfo(context.Background(), asset.Spot); err != nil {
		t.Error("Okx FetchAccountInfo() error", err)
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetAccountFundingHistory(context.Background()); err != nil {
		t.Error("Okx GetFundingHistory() error", err)
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var resp WsPlaceOrderInput
	err := json.Unmarshal([]byte(placeOrderArgs), &resp)
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Arguments) == 0 {
		t.Error("order not found")
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
	_, err = ok.SubmitOrder(context.Background(), orderSubmission)
	if err != nil {
		t.Error("Okx SubmitOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var orderCancellation = &order.Cancel{
		OrderID:       "1",
		WalletAddress: core.BitcoinDonationAddress,
		AccountID:     "1",
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		AssetType:     asset.Spot,
	}
	if err := ok.CancelOrder(context.Background(), orderCancellation); err != nil {
		t.Error(err)
	}
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

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
	if err != nil && !strings.Contains(err.Error(), "order does not exist.") {
		t.Error("Okx CancelBatchOrders() error", err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CancelAllOrders(context.Background(), &order.Cancel{}); err != nil {
		t.Errorf("%s CancelAllOrders() error: %v", ok.Name, err)
	}
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.ModifyOrder(context.Background(),
		&order.Modify{
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.LTC, currency.BTC),
			OrderID:   "1234",
			Price:     123456.44,
			Amount:    123,
		})
	if err != nil {
		t.Errorf("Okx ModifyOrder() error %v", err)
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	enabled, err := ok.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error("couldn't find enabled tradable pairs")
	}
	if len(enabled) == 0 {
		t.SkipNow()
	}
	_, err = ok.GetOrderInfo(context.Background(),
		"123", enabled[0], asset.Futures)
	if err != nil && !strings.Contains(err.Error(), "Order does not exist") {
		t.Errorf("Okx GetOrderInfo() expecting %s, but found %v", "Order does not exist", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetDepositAddress(context.Background(), currency.BTC, "", ""); err != nil && !errors.Is(err, errDepositAddressNotFound) {
		t.Error("Okx GetDepositAddress() error", err)
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	withdrawCryptoRequest := withdraw.Request{
		Exchange: ok.Name,
		Amount:   0.00000000001,
		Currency: currency.BTC,
		Crypto: withdraw.CryptoRequest{
			Address: core.BitcoinDonationAddress,
		},
	}
	if _, err := ok.WithdrawCryptocurrencyFunds(context.Background(), &withdrawCryptoRequest); err != nil {
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	pair, err := currency.NewPairFromString("BTC-USD")
	if err != nil {
		t.Error(err)
	}
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     currency.Pairs{pair, currency.NewPair(currency.USDT, currency.USD), currency.NewPair(currency.USD, currency.LTC)},
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	if _, err := ok.GetActiveOrders(context.Background(), &getOrdersRequest); err != nil {
		t.Error("Okx GetActiveOrders() error", err)
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

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

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.ValidateAPICredentials(context.Background(), asset.Spot); err != nil {
		t.Errorf("%s ValidateAPICredentials() error %v", ok.Name, err)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	startTime := time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 0, 100)
	_, err := ok.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.OneDay, startTime, endTime)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ok.GetHistoricCandles(context.Background(), pair, asset.Spot, kline.Interval(time.Hour*4), startTime, endTime)
	if !errors.Is(err, kline.ErrRequestExceedsExchangeLimits) {
		t.Errorf("received: '%v' but expected: '%v'", err, kline.ErrRequestExceedsExchangeLimits)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	currencyPair := currency.NewPair(currency.BTC, currency.USDT)
	_, err := ok.GetHistoricCandlesExtended(context.Background(), currencyPair, asset.Spot, kline.OneMin, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		t.Errorf("%s GetHistoricCandlesExtended() error: %v", ok.Name, err)
	}
}

const wsInstrumentPushData = `{"arg": {"channel": "instruments","instType": "FUTURES"},"data": [{"instType": "FUTURES","instId": "BTC-USD-191115","uly": "BTC-USD","category": "1","baseCcy": "","quoteCcy": "","settleCcy": "BTC","ctVal": "10","ctMult": "1","ctValCcy": "USD","optType": "","stk": "","listTime": "","expTime": "","tickSz": "0.01","lotSz": "1","minSz": "1","ctType": "linear","alias": "this_week","state": "live","maxLmtSz":"10000","maxMktSz":"99999","maxTwapSz":"99999","maxIcebergSz":"99999","maxTriggerSz":"9999","maxStopSz":"9999"}]}`

func TestWSInstruments(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(wsInstrumentPushData)); err != nil {
		t.Errorf("%s Websocket Instruments Push Data error %v", ok.Name, err)
	}
}

var tickerChannelPushData = `{"arg": {"channel": "tickers","instId": "%v"},"data": [{"instType": "SWAP","instId": "%v","last": "9999.99","lastSz": "0.1","askPx": "9999.99","askSz": "11","bidPx": "8888.88","bidSz": "5","open24h": "9000","high24h": "10000","low24h": "8888.88","volCcy24h": "2222","vol24h": "2222","sodUtc0": "2222","sodUtc8": "2222","ts": "1597026383085"}]}`

func TestTickerChannel(t *testing.T) {
	t.Parallel()
	curr := ok.CurrencyPairs.Pairs[asset.PerpetualSwap].Enabled[0]
	if err := ok.WsHandleData([]byte(fmt.Sprintf(tickerChannelPushData, curr, curr))); err != nil {
		t.Error("Okx TickerChannel push data error", err)
	}
}

const openInterestChannelPushData = `{"arg": {"channel": "open-interest","instId": "LTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","oi": "5000","oiCcy": "555.55","ts": "1597026383085"}]}`

func TestOpenInterestPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(openInterestChannelPushData)); err != nil {
		t.Error("Okx Open Interest Push Data error", err)
	}
}

var candlesticksPushData = `{"arg": {"channel": "candle1D","instId": "%v"},"data": [["1597026383085","8533.02","8553.74","8527.17","8548.26","45247","529.5858061"]]}`

func TestCandlestickPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(fmt.Sprintf(candlesticksPushData, ok.CurrencyPairs.Pairs[asset.Futures].Enabled[0]))); err != nil {
		t.Error("Okx Candlestick Push Data error", err)
	}
}

const tradePushDataJSON = `{"arg": {"channel": "trades","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","tradeId": "130639474","px": "42219.9","sz": "0.12060306","side": "buy","ts": "1630048897897"}]}`

func TestTradePushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(tradePushDataJSON)); err != nil {
		t.Error("Okx Trade Push Data error", err)
	}
}

const estimatedDeliveryAndExercisePricePushDataJSON = `{"arg": {"args": "estimated-price","instType": "FUTURES","uly": "BTC-USD"},"data": [{"instType": "FUTURES","instId": "BTC-USD-170310","settlePx": "200","ts": "1597026383085"}]}`

func TestEstimatedDeliveryAndExercisePricePushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(estimatedDeliveryAndExercisePricePushDataJSON)); err != nil {
		t.Error("Okx Estimated Delivery and Exercise Price Push Data error", err)
	}
}

const markPricePushData = `{"arg": {"channel": "mark-price","instId": "LTC-USD-190628"},"data": [{"instType": "FUTURES","instId": "LTC-USD-190628","markPx": "0.1","ts": "1597026383085"}]}`

func TestMarkPricePushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(markPricePushData)); err != nil {
		t.Error("Okx Mark Price Push Data error", err)
	}
}

const markPriceCandlestickPushData = `{"arg": {"channel": "mark-price-candle1D","instId": "BTC-USD-190628"},"data": [["1597026383085", "3.721", "3.743", "3.677", "3.708"],["1597026383085", "3.731", "3.799", "3.494", "3.72"]]}`

func TestMarkPriceCandlestickPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(markPriceCandlestickPushData)); err != nil {
		t.Error("Okx Mark Price Candlestick Push Data error", err)
	}
}

const priceLimitPushDataJSON = `{    "arg": {        "channel": "price-limit",        "instId": "LTC-USD-190628"    },    "data": [{        "instId": "LTC-USD-190628",        "buyLmt": "200",        "sellLmt": "300",        "ts": "1597026383085"    }]}`

func TestPriceLimitPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(priceLimitPushDataJSON)); err != nil {
		t.Error("Okx Price Limit Push Data error", err)
	}
}

const testSnapshotOrderbookPushData = `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`
const updateOrderBookPushDataJSON = `{"arg":{"channel":"books","instId":"BTC-USDT"},"action":"update","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`

func TestSnapshotAndUpdateOrderBookPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(testSnapshotOrderbookPushData)); err != nil {
		t.Error("Okx Snapshot order book push data error", err)
	}
	if err := ok.WsHandleData([]byte(updateOrderBookPushDataJSON)); err != nil {
		t.Error("Okx Update Order Book Push Data error", err)
	}
}

var snapshotOrderBookPushData = `{"arg":{"channel":"books","instId":"%v"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`

func TestSnapshotPushData(t *testing.T) {
	t.Parallel()
	err := ok.WsHandleData([]byte(fmt.Sprintf(snapshotOrderBookPushData, ok.CurrencyPairs.Pairs[asset.Futures].Enabled[0])))
	if err != nil {
		t.Error("Okx Snapshot order book push data error", err)
	}
}

const calculateOrderbookChecksumUpdateorderbookJSON = `{"Bids":[{"Amount":56,"Price":0.07014,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":608,"Price":0.07011,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":110,"Price":0.07009,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1264,"Price":0.07006,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":2347,"Price":0.07004,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":279,"Price":0.07003,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":52,"Price":0.07001,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":91,"Price":0.06997,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4242,"Price":0.06996,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":486,"Price":0.06995,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":161,"Price":0.06992,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":63,"Price":0.06991,"ID":0,"Period":0,"LiquidationOrders":0,
"OrderCount":0},{"Amount":7518,"Price":0.06988,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":186,"Price":0.06976,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":71,"Price":0.06975,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1086,"Price":0.06973,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":513,"Price":0.06961,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4603,"Price":0.06959,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":186,"Price":0.0695,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3043,"Price":0.06946,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":103,"Price":0.06939,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5053,"Price":0.0693,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5039,"Price":0.06909,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5037,"Price":0.06888,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1526,"Price":0.06886,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5008,"Price":0.06867,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5065,"Price":0.06846,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1572,"Price":0.06826,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1565,"Price":0.06801,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":67,"Price":0.06748,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":111,"Price":0.0674,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10038,"Price":0.0672,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.06652,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1526,"Price":0.06625,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10924,"Price":0.06619,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.05986,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.05387,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.04848,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.04363,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0}],"Asks":[{"Amount":5,"Price":0.07026,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":765,"Price":0.07027,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":110,"Price":0.07028,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1264,"Price":0.0703,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":280,"Price":0.07034,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":2255,"Price":0.07035,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":28,"Price":0.07036,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":63,"Price":0.07037,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":137,"Price":0.07039,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":48,"Price":0.0704,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":32,"Price":0.07041,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3985,"Price":0.07043,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":257,"Price":0.07057,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":7870,"Price":0.07058,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":161,"Price":0.07059,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4539,"Price":0.07061,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1438,"Price":0.07068,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3162,"Price":0.07088,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":99,"Price":0.07104,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5018,"Price":0.07108,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1540,"Price":0.07115,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5080,"Price":0.07129,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1512,"Price":0.07145,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5016,"Price":0.0715,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5026,"Price":0.07171,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5062,"Price":0.07192,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1517,"Price":0.07197,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1511,"Price":0.0726,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10376,"Price":0.07314,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.07354,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10277,"Price":0.07466,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":269,"Price":0.07626,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":269,"Price":0.07636,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.0809,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.08899,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.09789,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.10768,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0}],"Exchange":"Okx","Pair":"BTC-USDT","Asset":"spot","LastUpdated":"0001-01-01T00:00:00Z","LastUpdateID":0,"PriceDuplication":false,"IsFundingRate":false,"RestSnapshot":false,"IDAlignment":false}`

func TestCalculateUpdateOrderbookChecksum(t *testing.T) {
	t.Parallel()

	var orderbookBase orderbook.Base
	err := json.Unmarshal([]byte(calculateOrderbookChecksumUpdateorderbookJSON), &orderbookBase)
	if err != nil {
		t.Errorf("%s error while deserializing to orderbook.Base %v", ok.Name, err)
	}
	if err := ok.CalculateUpdateOrderbookChecksum(&orderbookBase, 2832680552); err != nil {
		t.Errorf("%s CalculateUpdateOrderbookChecksum() error: %v", ok.Name, err)
	}
}

const optionSummaryPushDataJSON = `{"arg": {"channel": "opt-summary","uly": "BTC-USD"},"data": [{"instType": "OPTION","instId": "BTC-USD-200103-5500-C","uly": "BTC-USD","delta": "0.7494223636","gamma": "-0.6765419039","theta": "-0.0000809873","vega": "0.0000077307","deltaBS": "0.7494223636","gammaBS": "-0.6765419039","thetaBS": "-0.0000809873","vegaBS": "0.0000077307","realVol": "0","bidVol": "","askVol": "1.5625","markVol": "0.9987","lever": "4.0342","fwdPx": "39016.8143629068452065","ts": "1597026383085"}]}`

func TestOptionSummaryPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(optionSummaryPushDataJSON)); err != nil {
		t.Error("Okx Option Summary Push Data error", err)
	}
}

const fundingRatePushDataJSON = `{"arg": {"channel": "funding-rate","instId": "BTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "BTC-USD-SWAP","fundingRate": "0.018","nextFundingRate": "","fundingTime": "1597026383085"}]}`

func TestFundingRatePushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(fundingRatePushDataJSON)); err != nil {
		t.Error("Okx Funding Rate Push Data error", err)
	}
}

var indexCandlestickPushDataJSON = `{"arg": {"channel": "index-candle30m","instId": "BTC-USDT"},"data": [["1597026383085", "3811.31", "3811.31", "3811.31", "3811.31"]]}`

func TestIndexCandlestickPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(indexCandlestickPushDataJSON)); err != nil {
		t.Error("Okx Index Candlestick Push Data error", err)
	}
}

const indexTickerPushDataJSON = `{"arg": {"channel": "index-tickers","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","idxPx": "0.1","high24h": "0.5","low24h": "0.1","open24h": "0.1","sodUtc0": "0.1","sodUtc8": "0.1","ts": "1597026383085"}]}`

func TestIndexTickersPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(indexTickerPushDataJSON)); err != nil {
		t.Error("Okx Index Ticker Push Data error", err)
	}
}

const statusPushDataJSON = `{"arg": {"channel": "status"},"data": [{"title": "Spot System Upgrade","state": "scheduled","begin": "1610019546","href": "","end": "1610019546","serviceType": "1","system": "classic","scheDesc": "","ts": "1597026383085"}]}`

func TestStatusPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(statusPushDataJSON)); err != nil {
		t.Error("Okx Status Push Data error", err)
	}
}

const publicStructBlockTradesPushDataJSON = `{"arg":{"channel":"public-struc-block-trades"},"data":[{"cTime":"1608267227834","blockTdId":"1802896","legs":[{"px":"0.323","sz":"25.0","instId":"BTC-USD-20220114-13250-C","side":"sell","tradeId":"15102"},{"px":"0.666","sz":"25","instId":"BTC-USD-20220114-21125-C","side":"buy","tradeId":"15103"}]}]}`

func TestPublicStructBlockTrades(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(publicStructBlockTradesPushDataJSON)); err != nil {
		t.Error("Okx Public Struct Block Trades error", err)
	}
}

const blockTickerPushDataJSON = `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`

func TestBlockTickerPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(blockTickerPushDataJSON)); err != nil {
		t.Error("Okx Block Tickers push data error", err)
	}
}

const accountPushDataJSON = `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`

func TestAccountPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(accountPushDataJSON)); err != nil {
		t.Error("Okx Account Push Data error", err)
	}
}

const positionPushDataJSON = `{"arg":{"channel":"positions","instType":"FUTURES"},"data":[{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`
const positionPushDataWithUnderlyingJSON = `{"arg": {"channel": "positions","uid": "77982378738415879","instType": "ANY"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`

func TestPositionPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(positionPushDataJSON)); err != nil {
		t.Error("Okx Account Push Data error", err)
	}
	if err := ok.WsHandleData([]byte(positionPushDataWithUnderlyingJSON)); err != nil {
		t.Error("Okx Account Push Data error", err)
	}
}

const balanceAndPositionJSON = `{"arg": {"channel": "balance_and_position","uid": "77982378738415879"},"data": [{"pTime": "1597026383085","eventType": "snapshot","balData": [{"ccy": "BTC","cashBal": "1","uTime": "1597026383085"}],"posData": [{"posId": "1111111111","tradeId": "2","instId": "BTC-USD-191018","instType": "FUTURES","mgnMode": "cross","posSide": "long","pos": "10","ccy": "BTC","posCcy": "","avgPx": "3320","uTIme": "1597026383085"}]}]}`

func TestBalanceAndPosition(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(balanceAndPositionJSON)); err != nil {
		t.Error("Okx Balance And Position error", err)
	}
}

const orderPushDataJSON = `{"arg": {    "channel": "orders",    "instType": "SPOT",    "instId": "BTC-USDT",    "uid": "614488474791936"},"data": [    {        "accFillSz": "0.001",        "amendResult": "",        "avgPx": "31527.1",        "cTime": "1654084334977",        "category": "normal",        "ccy": "",        "clOrdId": "",        "code": "0",        "execType": "M",        "fee": "-0.02522168",        "feeCcy": "USDT",        "fillFee": "-0.02522168",        "fillFeeCcy": "USDT",        "fillNotionalUsd": "31.50818374",        "fillPx": "31527.1",        "fillSz": "0.001",        "fillTime": "1654084353263",        "instId": "BTC-USDT",        "instType": "SPOT",        "lever": "0",        "msg": "",        "notionalUsd": "31.50818374",        "ordId": "452197707845865472",        "ordType": "limit",        "pnl": "0",        "posSide": "",        "px": "31527.1",        "rebate": "0",        "rebateCcy": "BTC",        "reduceOnly": "false",        "reqId": "",        "side": "sell",        "slOrdPx": "",        "slTriggerPx": "",        "slTriggerPxType": "last",        "source": "",        "state": "filled",        "sz": "0.001",        "tag": "",        "tdMode": "cash",        "tgtCcy": "",        "tpOrdPx": "",        "tpTriggerPx": "",        "tpTriggerPxType": "last",        "tradeId": "242589207",        "uTime": "1654084353264"    }]}`

func TestOrderPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(orderPushDataJSON)); err != nil {
		t.Error("Okx Order Push Data error", err)
	}
}

const algoOrdersPushDataJSON = `{"arg": {"channel": "orders-algo","uid": "77982378738415879","instType": "FUTURES","instId": "BTC-USD-200329"},"data": [{"instType": "FUTURES","instId": "BTC-USD-200329","ordId": "312269865356374016","ccy": "BTC","algoId": "1234","px": "999","sz": "3","tdMode": "cross","tgtCcy": "","notionalUsd": "","ordType": "trigger","side": "buy","posSide": "long","state": "live","lever": "20","tpTriggerPx": "","tpTriggerPxType": "","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "","triggerPx": "99","triggerPxType": "last","ordPx": "12","actualSz": "","actualPx": "","tag": "adadadadad","actualSide": "","triggerTime": "1597026383085","cTime": "1597026383000"}]}`

func TestAlgoOrderPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(algoOrdersPushDataJSON)); err != nil {
		t.Error("Okx Algo Order Push Data error", err)
	}
}

const advancedAlgoOrderPushDataJSON = `{"arg":{"channel":"algo-advance","uid": "77982378738415879","instType":"SPOT","instId":"BTC-USDT"},"data":[{"actualPx":"","actualSide":"","actualSz":"0","algoId":"355056228680335360","cTime":"1630924001545","ccy":"","count":"1","instId":"BTC-USDT","instType":"SPOT","lever":"0","notionalUsd":"","ordPx":"","ordType":"iceberg","pTime":"1630924295204","posSide":"net","pxLimit":"10","pxSpread":"1","pxVar":"","side":"buy","slOrdPx":"","slTriggerPx":"","state":"pause","sz":"0.1","szLimit":"0.1","tdMode":"cash","timeInterval":"","tpOrdPx":"","tpTriggerPx":"","tag": "adadadadad","triggerPx":"","triggerTime":"","callbackRatio":"","callbackSpread":"","activePx":"","moveTriggerPx":""}]}`

func TestAdvancedAlgoOrderPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(advancedAlgoOrderPushDataJSON)); err != nil {
		t.Error("Okx Advanced Algo Orders Push Data error", err)
	}
}

const positionRiskPushDataJSON = `{"arg": {"channel": "liquidation-warning","uid": "77982378738415879","instType": "ANY"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`

func TestPositionRiskPushDataJSON(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(positionRiskPushDataJSON)); err != nil {
		t.Error("Okx Position Risk Push Data error", err)
	}
}

const accountGreeksPushData = `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"",    "vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`

func TestAccountGreeksPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(accountGreeksPushData)); err != nil {
		t.Error("Okx Account Greeks Push Data error", err)
	}
}

const rfqsPushDataJSON = `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"",    "vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`

func TestRfqs(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(rfqsPushDataJSON)); err != nil {
		t.Error("Okx RFQS Push Data error", err)
	}
}

const accountsPushDataJSON = `{	"arg": {	  "channel": "account",	  "ccy": "BTC",	  "uid": "77982378738415879"	},	"data": [	  {		"uTime": "1597026383085",		"totalEq": "41624.32",		"isoEq": "3624.32",		"adjEq": "41624.32",		"ordFroz": "0",		"imr": "4162.33",		"mmr": "4",		"notionalUsd": "",		"mgnRatio": "41624.32",		"details": [		  {			"availBal": "",			"availEq": "1",			"ccy": "BTC",			"cashBal": "1",			"uTime": "1617279471503",			"disEq": "50559.01",			"eq": "1",			"eqUsd": "45078.3790756226851775",			"frozenBal": "0",			"interest": "0",			"isoEq": "0",			"liab": "0",			"maxLoan": "",			"mgnRatio": "",			"notionalLever": "0.0022195262185864",			"ordFrozen": "0",			"upl": "0",			"uplLiab": "0",			"crossLiab": "0",			"isoLiab": "0",			"coinUsdPrice": "60000",			"stgyEq":"0",			"spotInUseAmt":"",			"isoUpl":""		  }		]	  }	]}`

func TestAccounts(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(accountsPushDataJSON)); err != nil {
		t.Errorf("%s Accounts push data error %v", ok.Name, err)
	}
}

const quotesPushDataJSON = `{"arg":{"channel":"quotes"},"data":[{"validUntil":"1608997227854","uTime":"1608267227834","cTime":"1608267227834","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-220114-25000-C","side":"sell","tgtCcy":""},{"px":"0.0045","sz":"25","instId":"BTC-USD-220114-35000-C","side":"buy","tgtCcy":""}],"quoteId":"25092","rfqId":"18753","traderCode":"SATS","quoteSide":"sell","state":"canceled","clQuoteId":""}]}`

func TestQuotesPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(quotesPushDataJSON)); err != nil {
		t.Error("Okx Quotes Push Data error", err)
	}
}

const structureBlockTradesPushDataJSON = `{"arg":{"channel":"struc-block-trades"},"data":[{"cTime":"1608267227834","rfqId":"18753","clRfqId":"","quoteId":"25092","clQuoteId":"","blockTdId":"180184","tTraderCode":"ANAND","mTraderCode":"WAGMI","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-20220630-60000-C","side":"sell","fee":"0.1001","feeCcy":"BTC","tradeId":"10211","tgtCcy":""},{"px":"0.0033","sz":"25","instId":"BTC-USD-20220630-50000-C","side":"buy","fee":"0.1001","feeCcy":"BTC","tradeId":"10212","tgtCcy":""}]}]}`

func TestStructureBlockTradesPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(structureBlockTradesPushDataJSON)); err != nil {
		t.Error("Okx Structure Block Trades error", err)
	}
}

const spotGridAlgoOrdersPushDataJSON = `{"arg": {"channel": "grid-orders-spot","instType": "ANY"},"data": [{"algoId": "448965992920907776","algoOrdType": "grid","annualizedRate": "0","arbitrageNum": "0","baseSz": "0","cTime": "1653313834104","cancelType": "0","curBaseSz": "0.001776289214","curQuoteSz": "46.801755866","floatProfit": "-0.4953878967772","gridNum": "6","gridProfit": "0","instId": "BTC-USDC","instType": "SPOT","investment": "100","maxPx": "33444.8","minPx": "24323.5","pTime": "1653476023742","perMaxProfitRate": "0.060375293181491054543","perMinProfitRate": "0.0455275366818586","pnlRatio": "0","quoteSz": "100","runPx": "30478.1","runType": "1","singleAmt": "0.00059261","slTriggerPx": "","state": "running","stopResult": "0","stopType": "0","totalAnnualizedRate": "-0.9643551057262827","totalPnl": "-0.4953878967772","tpTriggerPx": "","tradeNum": "3","triggerTime": "1653378736894","uTime": "1653378736894"}]}`

func TestSpotGridAlgoOrdersPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(spotGridAlgoOrdersPushDataJSON)); err != nil {
		t.Error("Okx Spot Grid Algo Orders Push Data error", err)
	}
}

const contractGridAlgoOrdersPushDataJSON = `{"arg": {"channel": "grid-orders-contract","instType": "ANY"},"data": [{"actualLever": "1.02","algoId": "449327675342323712","algoOrdType": "contract_grid","annualizedRate": "0.7572437878956523","arbitrageNum": "1","basePos": true,"cTime": "1653400065912","cancelType": "0","direction": "long","eq": "10129.419829834853","floatProfit": "109.537858234853","gridNum": "50","gridProfit": "19.8819716","instId": "BTC-USDT-SWAP","instType": "SWAP","investment": "10000","lever": "5","liqPx": "603.2149534767834","maxPx": "100000","minPx": "10","pTime": "1653484573918","perMaxProfitRate": "995.7080916791230692","perMinProfitRate": "0.0946277854875634","pnlRatio": "0.0129419829834853","runPx": "29216.3","runType": "1","singleAmt": "1","slTriggerPx": "","state": "running","stopType": "0","sz": "10000","tag": "","totalAnnualizedRate": "4.929207431970923","totalPnl": "129.419829834853","tpTriggerPx": "","tradeNum": "37","triggerTime": "1653400066940","uTime": "1653484573589","uly": "BTC-USDT"}]}`

func TestContractGridAlgoOrdersPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(contractGridAlgoOrdersPushDataJSON)); err != nil {
		t.Error("Okx Contract Grid Algo Order Push Data error", err)
	}
}

const gridPositionsPushDataJSON = `{"arg": {"channel": "grid-positions","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"adl": "1","algoId": "449327675342323712","avgPx": "29181.4638888888888895","cTime": "1653400065917","ccy": "USDT","imr": "2089.2690000000002","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29852.7","lever": "5","liqPx": "604.7617536513744","markPx": "29849.7","mgnMode": "cross","mgnRatio": "217.71740878394456","mmr": "41.78538","notionalUsd": "10435.794191550001","pTime": "1653536068723","pos": "35","posSide": "net","uTime": "1653445498682","upl": "232.83263888888962","uplRatio": "0.1139826489932205"}]}`

func TestGridPositionsPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(gridPositionsPushDataJSON)); err != nil {
		t.Error("Okx Grid Positions Push Data error", err)
	}
}

const gridSubOrdersPushDataJSON = `{"arg": {"channel": "grid-sub-orders","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"accFillSz": "0","algoId": "449327675342323712","algoOrdType": "contract_grid","avgPx": "0","cTime": "1653445498664","ctVal": "0.01","fee": "0","feeCcy": "USDT","groupId": "-1","instId": "BTC-USDT-SWAP","instType": "SWAP","lever": "5","ordId": "449518234142904321","ordType": "limit","pTime": "1653486524502","pnl": "","posSide": "net","px": "28007.2","side": "buy","state": "live","sz": "1","tag":"","tdMode": "cross","uTime": "1653445498674"}]}`

func TestGridSubOrdersPushData(t *testing.T) {
	t.Parallel()
	if err := ok.WsHandleData([]byte(gridSubOrdersPushDataJSON)); err != nil {
		t.Error("Okx Grid Sub orders Push Data error", err)
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetHistoricTrades(context.Background(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot, time.Now().Add(-time.Minute*4), time.Now().Add(-time.Minute*2)); err != nil {
		t.Errorf("%s GetHistoricTrades() error %v", ok.Name, err)
	}
}

func setupWS() {
	if !ok.Websocket.IsEnabled() {
		return
	}
	if !sharedtestvalues.AreAPICredentialsSet(ok) {
		ok.Websocket.SetCanUseAuthenticatedEndpoints(false)
	}
	err := ok.WsConnect()
	if err != nil {
		log.Fatal(err)
	}
}

// ************************** Public Channel Subscriptions *****************************

func TestInstrumentsSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.InstrumentsSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s InstrumentsSubscription() error: %v", ok.Name, err)
	}
}

func TestTickersSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.TickersSubscription("subscribe", asset.Margin, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s TickersSubscription() error: %v", ok.Name, err)
	}
	if err := ok.TickersSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s TickersSubscription() error: %v", ok.Name, err)
	}
}
func TestOpenInterestSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.OpenInterestSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.BTC, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s OpenInterestSubscription() error: %v", ok.Name, err)
	}
}
func TestCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.PerpetualSwap)
	if err != nil {
		t.Error("couldn't find enabled tradable pairs")
	}
	if len(enabled) == 0 {
		t.SkipNow()
	}
	if err := ok.CandlesticksSubscription("subscribe", okxChannelCandle1m, asset.Futures, enabled[0]); err != nil {
		t.Errorf("%s CandlesticksSubscription() error: %v", ok.Name, err)
	}
}

func TestTradesSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.TradesSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s TradesSubscription() error: %v", ok.Name, err)
	}
}

func TestEstimatedDeliveryExercisePriceSubscription(t *testing.T) {
	t.Parallel()
	futuresPairs, err := ok.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Errorf("%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	}
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	if err := ok.EstimatedDeliveryExercisePriceSubscription("subscribe", asset.Futures, futuresPairs[0]); err != nil {
		t.Errorf("%s EstimatedDeliveryExercisePriceSubscription() error: %v", ok.Name, err)
	}
}

func TestMarkPriceSubscription(t *testing.T) {
	t.Parallel()
	futuresPairs, err := ok.FetchTradablePairs(context.Background(), asset.Futures)
	if err != nil {
		t.Errorf("%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	}
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	if err := ok.MarkPriceSubscription("subscribe", asset.Futures, futuresPairs[0]); err != nil {
		t.Errorf("%s MarkPriceSubscription() error: %v", ok.Name, err)
	}
}

func TestMarkPriceCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error("couldn't find enabled tradable pairs")
	}
	if len(enabled) == 0 {
		t.SkipNow()
	}
	if err := ok.MarkPriceCandlesticksSubscription("subscribe", okxChannelMarkPriceCandle1Y, asset.Futures, enabled[0]); err != nil {
		t.Errorf("%s MarkPriceCandlesticksSubscription() error: %v", ok.Name, err)
	}
}

func TestPriceLimitSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.PriceLimitSubscription("subscribe", currency.Pair{Base: currency.NewCode("BTC"), Quote: currency.NewCode("USDT-SWAP")}); err != nil {
		t.Errorf("%s PriceLimitSubscription() error: %v", ok.Name, err)
	}
}

func TestOrderBooksSubscription(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.Spot)
	if err != nil {
		t.Error("couldn't find enabled tradable pairs")
	}
	if len(enabled) == 0 {
		t.SkipNow()
	}
	if err := ok.OrderBooksSubscription("subscribe", okxChannelOrderBooks, asset.Futures, enabled[0]); err != nil {
		t.Errorf("%s OrderBooksSubscription() error: %v", ok.Name, err)
	}
	if err := ok.OrderBooksSubscription("unsubscribe", okxChannelOrderBooks, asset.Futures, enabled[0]); err != nil {
		t.Errorf("%s OrderBooksSubscription() error: %v", ok.Name, err)
	}
}

func TestOptionSummarySubscription(t *testing.T) {
	t.Parallel()
	if err := ok.OptionSummarySubscription("subscribe", currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s OptionSummarySubscription() error: %v", ok.Name, err)
	}
	if err := ok.OptionSummarySubscription("unsubscribe", currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s OptionSummarySubscription() error: %v", ok.Name, err)
	}
}

func TestFundingRateSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.FundingRateSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP"))); err != nil {
		t.Errorf("%s FundingRateSubscription() error: %v", ok.Name, err)
	}
	if err := ok.FundingRateSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP"))); err != nil {
		t.Errorf("%s FundingRateSubscription() error: %v", ok.Name, err)
	}
}

func TestIndexCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.IndexCandlesticksSubscription("subscribe", okxChannelIndexCandle6M, asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s IndexCandlesticksSubscription() error: %v", ok.Name, err)
	}
	if err := ok.IndexCandlesticksSubscription("unsubscribe", okxChannelIndexCandle6M, asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s IndexCandlesticksSubscription() error: %v", ok.Name, err)
	}
}
func TestIndexTickerChannelIndexTickerChannel(t *testing.T) {
	t.Parallel()
	if err := ok.IndexTickerChannel("subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s IndexTickerChannel() error: %v", ok.Name, err)
	}
	if err := ok.IndexTickerChannel("unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s IndexTickerChannel() error: %v", ok.Name, err)
	}
}

func TestStatusSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.StatusSubscription("subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s StatusSubscription() error: %v", ok.Name, err)
	}
	if err := ok.StatusSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s StatusSubscription() error: %v", ok.Name, err)
	}
}

func TestPublicStructureBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.PublicStructureBlockTradesSubscription("subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s PublicStructureBlockTradesSubscription() error: %v", ok.Name, err)
	}
	if err := ok.PublicStructureBlockTradesSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s PublicStructureBlockTradesSubscription() error: %v", ok.Name, err)
	}
}
func TestBlockTickerSubscription(t *testing.T) {
	t.Parallel()
	if err := ok.BlockTickerSubscription("subscribe", asset.Options, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s BlockTickerSubscription() error: %v", ok.Name, err)
	}
	if err := ok.BlockTickerSubscription("unsubscribe", asset.Options, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s BlockTickerSubscription() error: %v", ok.Name, err)
	}
}

// ************ Authenticated Websocket endpoints Test **********************************************

func TestWsAccountSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.WsAccountSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT)); err != nil {
		t.Errorf("%s WsAccountSubscription() error: %v", ok.Name, err)
	}
}

const placeOrderJSON = `{	"id": "1512",	"op": "order",	"args": [{ "instId":"BTC-USDC",    "tdMode":"cash",    "clOrdId":"b15",    "side":"Buy",    "ordType":"limit",    "px":"2.15",    "sz":"2"}	]}`

func TestWsPlaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var resp WsPlaceOrderInput
	err := json.Unmarshal([]byte(placeOrderArgs), &resp)
	if err != nil {
		t.Error(err)
	}
	var response OrderData
	err = json.Unmarshal([]byte(placeOrderJSON), &response)
	if err != nil {
		t.Error(err)
	}

	if _, err := ok.WsPlaceOrder(&PlaceOrderRequestParam{
		InstrumentID: "BTC-USDC",
		TradeMode:    "cross",
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       2.6,
		Price:        2.1,
		Currency:     "BTC",
	}); err != nil {
		t.Errorf("%s WsPlaceOrder() error: %v", ok.Name, err)
	}
}

const placeOrderArgs = `{	"id": "1513",	"op": "batch-orders",	"args": [	  {		"side": "buy",		"instId": "BTC-USDT",		"tdMode": "cash",		"ordType": "market",		"sz": "100"	  },	  {		"side": "buy",		"instId": "LTC-USDT",		"tdMode": "cash",		"ordType": "market",		"sz": "1"	  }	]}`

func TestWsPlaceMultipleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var resp WsPlaceOrderInput
	if err := json.Unmarshal([]byte(placeOrderArgs), &resp); err != nil {
		t.Error(err)
	}
	pairs, err := ok.FetchTradablePairs(context.Background(), asset.Spot)
	if err != nil {
		t.Fatal(err)
	} else if len(pairs) == 0 {
		t.Skip("no pairs found")
	}
	if _, err := ok.WsPlaceMultipleOrder(resp.Arguments); err != nil {
		t.Error("Okx WsPlaceMultipleOrder() error", err)
	}
}

func TestWsCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.WsCancelOrder(CancelOrderRequestParam{
		InstrumentID: "BTC-USD-190927",
		OrderID:      "2510789768709120",
	}); err != nil {
		t.Error("Okx WsCancelOrder() error", err)
	}
}

func TestWsCancleMultipleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.WsCancelMultipleOrder([]CancelOrderRequestParam{{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
	}}); err != nil && !strings.Contains(err.Error(), "Cancellation failed as the order does not exist.") {
		t.Error("Okx WsCancleMultipleOrder() error", err)
	}
}

func TestWsAmendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.WsAmendMultipleOrders([]AmendOrderRequestParams{
		{
			InstrumentID: "DCR-BTC",
			OrderID:      "2510789768709120",
			NewPrice:     1233324.332,
			NewQuantity:  1234,
		},
	}); err != nil && !strings.Contains(err.Error(), "Order modification failed as the order does not exist.") {
		t.Errorf("%s WsAmendMultipleOrders() %v", ok.Name, err)
	}
}

func TestWsPositionChannel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.WsPositionChannel("subscribe", asset.Options, currency.NewPair(currency.USD, currency.BTC)); err != nil {
		t.Errorf("%s WsPositionChannel() error : %v", ok.Name, err)
	}
}

func TestBalanceAndPositionSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.BalanceAndPositionSubscription("subscribe", "1234"); err != nil {
		t.Errorf("%s BalanceAndPositionSubscription() error %v", ok.Name, err)
	}
	if err := ok.BalanceAndPositionSubscription("unsubscribe", "1234"); err != nil {
		t.Errorf("%s BalanceAndPositionSubscription() error %v", ok.Name, err)
	}
}

func TestWsOrderChannel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.WsOrderChannel("subscribe", asset.Margin, currency.NewPair(currency.SOL, currency.USDT), ""); err != nil {
		t.Errorf("%s WsOrderChannel() error: %v", ok.Name, err)
	}
	if err := ok.WsOrderChannel("unsubscribe", asset.Margin, currency.NewPair(currency.SOL, currency.USDT), ""); err != nil {
		t.Errorf("%s WsOrderChannel() error: %v", ok.Name, err)
	}
}

func TestAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.AlgoOrdersSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s AlgoOrdersSubscription() error: %v", ok.Name, err)
	}
	if err := ok.AlgoOrdersSubscription("unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s AlgoOrdersSubscription() error: %v", ok.Name, err)
	}
}

func TestAdvanceAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.AdvanceAlgoOrdersSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")), ""); err != nil {
		t.Errorf("%s AdvanceAlgoOrdersSubscription() error: %v", ok.Name, err)
	}
	if err := ok.AdvanceAlgoOrdersSubscription("unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")), ""); err != nil {
		t.Errorf("%s AdvanceAlgoOrdersSubscription() error: %v", ok.Name, err)
	}
}

func TestPositionRiskWarningSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.PositionRiskWarningSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s PositionRiskWarningSubscription() error: %v", ok.Name, err)
	}
	if err := ok.PositionRiskWarningSubscription("unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP"))); err != nil {
		t.Errorf("%s PositionRiskWarningSubscription() error: %v", ok.Name, err)
	}
}

func TestAccountGreeksSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.AccountGreeksSubscription("subscribe", currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s AccountGreeksSubscription() error: %v", ok.Name, err)
	}
	if err := ok.AccountGreeksSubscription("unsubscribe", currency.NewPair(currency.SOL, currency.USD)); err != nil {
		t.Errorf("%s AccountGreeksSubscription() error: %v", ok.Name, err)
	}
}

func TestRfqSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.RfqSubscription("subscribe", ""); err != nil {
		t.Errorf("%s RfqSubscription() error: %v", ok.Name, err)
	}
	if err := ok.RfqSubscription("unsubscribe", ""); err != nil {
		t.Errorf("%s RfqSubscription() error: %v", ok.Name, err)
	}
}

func TestQuotesSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.QuotesSubscription("subscribe"); err != nil {
		t.Errorf("%s QuotesSubscription() error: %v", ok.Name, err)
	}
	if err := ok.QuotesSubscription("unsubscribe"); err != nil {
		t.Errorf("%s QuotesSubscription() error: %v", ok.Name, err)
	}
}

func TestStructureBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.StructureBlockTradesSubscription("subscribe"); err != nil {
		t.Errorf("%s StructureBlockTradesSubscription() error: %v", ok.Name, err)
	}
	if err := ok.StructureBlockTradesSubscription("unsubscribe"); err != nil {
		t.Errorf("%s StructureBlockTradesSubscription() error: %v", ok.Name, err)
	}
}

func TestSpotGridAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.SpotGridAlgoOrdersSubscription("subscribe", asset.Empty, currency.EMPTYPAIR, ""); err != nil {
		t.Errorf("%s SpotGridAlgoOrdersSubscription() error: %v", ok.Name, err)
	}
	if err := ok.SpotGridAlgoOrdersSubscription("unsubscribe", asset.Empty, currency.EMPTYPAIR, ""); err != nil {
		t.Errorf("%s SpotGridAlgoOrdersSubscription() error: %v", ok.Name, err)
	}
}

func TestContractGridAlgoOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.ContractGridAlgoOrders("subscribe", asset.Empty, currency.EMPTYPAIR, ""); err != nil {
		t.Errorf("%s ContractGridAlgoOrders() error: %v", ok.Name, err)
	}
	if err := ok.ContractGridAlgoOrders("unsubscribe", asset.Empty, currency.EMPTYPAIR, ""); err != nil {
		t.Errorf("%s ContractGridAlgoOrders() error: %v", ok.Name, err)
	}
}

func TestGridPositionsSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.GridPositionsSubscription("subscribe", "1234"); err != nil && !strings.Contains(err.Error(), "channel:grid-positions doesn't exist") {
		t.Errorf("%s GridPositionsSubscription() error: %v", ok.Name, err)
	}
	if err := ok.GridPositionsSubscription("unsubscribe", "1234"); err != nil && !strings.Contains(err.Error(), "channel:grid-positions doesn't exist") {
		t.Errorf("%s GridPositionsSubscription() error: %v", ok.Name, err)
	}
}

func TestGridSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if err := ok.GridSubOrders("subscribe", ""); err != nil && !strings.Contains(err.Error(), "grid-sub-orders doesn't exist") {
		t.Errorf("%s GridSubOrders() error: %v", ok.Name, err)
	}
	if err := ok.GridSubOrders("unsubscribe", ""); err != nil && !strings.Contains(err.Error(), "grid-sub-orders doesn't exist") {
		t.Errorf("%s GridSubOrders() error: %v", ok.Name, err)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	if _, err := ok.GetServerTime(context.Background(), asset.Empty); err != nil {
		t.Error(err)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetAvailableTransferChains(context.Background(), currency.BTC); err != nil {
		t.Error(err)
	}
}

func TestGetIntervalEnum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Description string
		Interval    kline.Interval
		Expected    string
		AppendUTC   bool
	}{
		{Description: "4hr with UTC", Interval: kline.FourHour, Expected: "4H", AppendUTC: true},
		{Description: "6H without UTC", Interval: kline.SixHour, Expected: "6H"},
		{Description: "6H with UTC", Interval: kline.SixHour, Expected: "6Hutc", AppendUTC: true},
		{Description: "Unsupported interval with UTC", Expected: "", AppendUTC: true},
	}

	for x := range tests {
		tt := tests[x]
		t.Run(tt.Description, func(t *testing.T) {
			t.Parallel()

			if r := ok.GetIntervalEnum(tt.Interval, tt.AppendUTC); r != tt.Expected {
				t.Errorf("%s: received: %s but expected: %s", tt.Description, r, tt.Expected)
			}
		})
	}
}

const instrumentJSON = `{"alias":"","baseCcy":"","category":"1","ctMult":"1","ctType":"linear","ctVal":"0.0001","ctValCcy":"BTC","expTime":"","instFamily":"BTC-USDC","instId":"BTC-USDC-SWAP","instType":"SWAP","lever":"125","listTime":"1666076190000","lotSz":"1","maxIcebergSz":"100000000.0000000000000000","maxLmtSz":"100000000","maxMktSz":"85000","maxStopSz":"85000","maxTriggerSz":"100000000.0000000000000000","maxTwapSz":"","minSz":"1","optType":"","quoteCcy":"","settleCcy":"USDC","state":"live","stk":"","tickSz":"0.1","uly":"BTC-USDC"}`

func TestInstrument(t *testing.T) {
	t.Parallel()

	var i Instrument
	err := json.Unmarshal([]byte(instrumentJSON), &i)
	if err != nil {
		t.Error(err)
	}

	if i.Alias != "" {
		t.Error("expected empty alias")
	}
	if i.BaseCurrency != "" {
		t.Error("expected empty base currency")
	}
	if i.Category != "1" {
		t.Error("expected 1 category")
	}
	if i.ContractMultiplier != "1" {
		t.Error("expected 1 contract multiplier")
	}
	if i.ContractType != "linear" {
		t.Error("expected linear contract type")
	}
	if i.ContractValue != "0.0001" {
		t.Error("expected 0.0001 contract value")
	}
	if i.ContractValueCurrency != currency.BTC.String() {
		t.Error("expected BTC contract value currency")
	}
	if !i.ExpTime.IsZero() {
		t.Error("expected empty expiry time")
	}
	if i.InstrumentFamily != "BTC-USDC" {
		t.Error("expected BTC-USDC instrument family")
	}
	if i.InstrumentID != "BTC-USDC-SWAP" {
		t.Error("expected BTC-USDC-SWAP instrument ID")
	}
	if i.InstrumentType != asset.PerpetualSwap {
		t.Error("expected SWAP instrument type")
	}
	if i.MaxLeverage != 125 {
		t.Error("expected 125 leverage")
	}
	if i.ListTime.UnixMilli() != 1666076190000 {
		t.Error("expected 1666076190000 listing time")
	}
	if i.LotSize != 1 {
		t.Error("expected 1 lot size")
	}
	if i.MaxSpotIcebergSize != 100000000.0000000000000000 {
		t.Error("expected 100000000.0000000000000000 max iceberg order size")
	}
	if i.MaxQuantityOfSpotLimitOrder != 100000000 {
		t.Error("expected 100000000 max limit order size")
	}
	if i.MaxQuantityOfMarketLimitOrder != 85000 {
		t.Error("expected 85000 max market order size")
	}
	if i.MaxStopSize != 85000 {
		t.Error("expected 85000 max stop order size")
	}
	if i.MaxTriggerSize != 100000000.0000000000000000 {
		t.Error("expected 100000000.0000000000000000 max trigger order size")
	}
	if i.MaxQuantityOfSpotTwapLimitOrder != 0 {
		t.Error("expected empty max TWAP size")
	}
	if i.MinimumOrderSize != 1 {
		t.Error("expected 1 min size")
	}
	if i.OptionType != "" {
		t.Error("expected empty option type")
	}
	if i.QuoteCurrency != "" {
		t.Error("expected empty quote currency")
	}
	if i.SettlementCurrency != currency.USDC.String() {
		t.Error("expected USDC settlement currency")
	}
	if i.State != "live" {
		t.Error("expected live state")
	}
	if i.StrikePrice != "" {
		t.Error("expected empty strike price")
	}
	if i.TickSize != 0.1 {
		t.Error("expected 0.1 tick size")
	}
	if i.Underlying != "BTC-USDC" {
		t.Error("expected BTC-USDC underlying")
	}
}

func TestGetLatestFundingRate(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD-SWAP")
	if err != nil {
		t.Error(err)
	}
	_, err = ok.GetLatestFundingRate(contextGenerate(), &fundingrate.LatestRateRequest{
		Asset:                asset.PerpetualSwap,
		Pair:                 cp,
		IncludePredictedRate: true,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetFundingRates(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD-SWAP")
	if err != nil {
		t.Error(err)
	}
	r := &fundingrate.RatesRequest{
		Asset:                asset.PerpetualSwap,
		Pair:                 cp,
		PaymentCurrency:      currency.USDT,
		StartDate:            time.Now().Add(-time.Hour * 24 * 7),
		EndDate:              time.Now(),
		IncludePredictedRate: true,
	}
	if sharedtestvalues.AreAPICredentialsSet(ok) {
		r.IncludePayments = true
	}
	_, err = ok.GetFundingRates(contextGenerate(), r)
	if err != nil {
		t.Error(err)
	}

	r.StartDate = time.Now().Add(-time.Hour * 24 * 120)
	_, err = ok.GetFundingRates(contextGenerate(), r)
	if !errors.Is(err, fundingrate.ErrFundingRateOutsideLimits) {
		t.Error(err)
	}

	r.RespectHistoryLimits = true
	_, err = ok.GetFundingRates(contextGenerate(), r)
	if err != nil {
		t.Error(err)
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := ok.IsPerpetualFutureCurrency(asset.Binary, currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error(err)
	}
	if is {
		t.Error("expected false")
	}

	cp, err := currency.NewPairFromString("BTC-USD-SWAP")
	if err != nil {
		t.Error(err)
	}
	is, err = ok.IsPerpetualFutureCurrency(asset.PerpetualSwap, cp)
	if err != nil {
		t.Error(err)
	}
	if !is {
		t.Error("expected true")
	}
}

func TestGetAssetsFromInstrumentTypeOrID(t *testing.T) {
	t.Parallel()
	_, err := ok.GetAssetsFromInstrumentTypeOrID("", "")
	if !errors.Is(err, errEmptyArgument) {
		t.Error(err)
	}

	assets, err := ok.GetAssetsFromInstrumentTypeOrID("SPOT", "")
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(assets) != 1 {
		t.Errorf("received %v expected %v", len(assets), 1)
	}
	if assets[0] != asset.Spot {
		t.Errorf("received %v expected %v", assets[0], asset.Spot)
	}

	assets, err = ok.GetAssetsFromInstrumentTypeOrID("", ok.CurrencyPairs.Pairs[asset.Futures].Enabled[0].String())
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(assets) != 1 {
		t.Errorf("received %v expected %v", len(assets), 1)
	}
	if assets[0] != asset.Futures {
		t.Errorf("received %v expected %v", assets[0], asset.Futures)
	}

	assets, err = ok.GetAssetsFromInstrumentTypeOrID("", ok.CurrencyPairs.Pairs[asset.PerpetualSwap].Enabled[0].String())
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if len(assets) != 1 {
		t.Errorf("received %v expected %v", len(assets), 1)
	}
	if assets[0] != asset.PerpetualSwap {
		t.Errorf("received %v expected %v", assets[0], asset.PerpetualSwap)
	}

	_, err = ok.GetAssetsFromInstrumentTypeOrID("", "test")
	if !errors.Is(err, currency.ErrCurrencyNotSupported) {
		t.Error(err)
	}

	_, err = ok.GetAssetsFromInstrumentTypeOrID("", "test-test")
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Error(err)
	}

	assets, err = ok.GetAssetsFromInstrumentTypeOrID("", ok.CurrencyPairs.Pairs[asset.Margin].Enabled[0].String())
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	var found bool
	for i := range assets {
		if assets[i] == asset.Margin {
			found = true
		}
	}
	if !found {
		t.Errorf("received %v expected %v", assets, asset.Margin)
	}

	assets, err = ok.GetAssetsFromInstrumentTypeOrID("", ok.CurrencyPairs.Pairs[asset.Spot].Enabled[0].String())
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	found = false
	for i := range assets {
		if assets[i] == asset.Spot {
			found = true
		}
	}
	if !found {
		t.Errorf("received %v expected %v", assets, asset.Spot)
	}
}
