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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
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
	err = ok.UpdateTradablePairs(contextGenerate(), true)
	if err != nil {
		log.Fatal(err)
	}
	if !useTestNet {
		ok.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
		ok.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
		setupWS()
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

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTickers(contextGenerate(), "OPTION", "", "SOL-USD")
	assert.NoError(t, err, "Okx GetTickers() error", err)
}

func TestGetIndexTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.GetIndexTickers(contextGenerate(), "USDT", "NEAR-USDT-SWAP")
	assert.NoError(t, err, "OKX GetIndexTicker() error", err)
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTicker(contextGenerate(), "NEAR-USDT-SWAP")
	assert.NoError(t, err)
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOrderBookDepth(contextGenerate(), "BTC-USDT", 400)
	assert.NoError(t, err)
}

func TestGetOrderBooksLite(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOrderBooksLite(contextGenerate(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCandlesticks(contextGenerate(), "BTC-USDT", kline.OneHour, time.Now().Add(-time.Minute*2), time.Now(), 2)
	assert.NoError(t, err)
}

func TestGetCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetCandlesticksHistory(contextGenerate(), "BTC-USDT", kline.OneHour, time.Unix(time.Now().Unix()-int64(time.Minute), 3), time.Now(), 3)
	assert.NoError(t, err)
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTrades(contextGenerate(), "BTC-USDT", 3)
	assert.NoError(t, err)
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTradesHistory(contextGenerate(), "BTC-USDT", "", "", 2)
	assert.NoError(t, err)
}

func TestGetoptionTradesByInstrumentFamily(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOptionTradesByInstrumentFamily(context.Background(), "BTC-USD")
	assert.NoError(t, err)
}

func TestGetOptionTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOptionTrades(context.Background(), "", "BTC-USD", "C")
	assert.NoError(t, err)
}

func TestGet24HTotalVolume(t *testing.T) {
	t.Parallel()
	_, err := ok.Get24HTotalVolume(contextGenerate())
	assert.NoError(t, err)
}

func TestGetOracle(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOracle(contextGenerate())
	assert.NoError(t, err)
}

func TestGetExchangeRate(t *testing.T) {
	t.Parallel()
	_, err := ok.GetExchangeRate(contextGenerate())
	assert.NoError(t, err)
}

func TestGetIndexComponents(t *testing.T) {
	t.Parallel()
	_, err := ok.GetIndexComponents(contextGenerate(), "ETH-USDT")
	assert.NoError(t, err)
}

func TestGetBlockTickers(t *testing.T) {
	t.Parallel()
	_, err := ok.GetBlockTickers(contextGenerate(), "SWAP", "")
	assert.NoError(t, err)
}

func TestGetBlockTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.GetBlockTicker(contextGenerate(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetBlockTrade(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicBlockTrades(contextGenerate(), "BTC-USDT")
	assert.NoError(t, err)
}

func TestGetInstrument(t *testing.T) {
	t.Parallel()
	_, err := ok.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: "OPTION",
		Underlying:     "SOL-USD",
	})
	assert.NoError(t, err)
	_, err = ok.GetInstruments(contextGenerate(), &InstrumentsFetchParams{
		InstrumentType: "OPTION",
		Underlying:     "SOL-USD",
	})
	assert.NoError(t, err)
}

func TestGetDeliveryHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetDeliveryHistory(contextGenerate(), "FUTURES", "BTC-USDT", time.Time{}, time.Time{}, 3)
	assert.NoError(t, err)
}

func TestGetOpenInterestData(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOpenInterestData(contextGenerate(), "FUTURES", "BTC-USDT", "")
	assert.NoError(t, err)
}

func TestGetSingleFundingRate(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSingleFundingRate(context.Background(), "BTC-USD-SWAP")
	assert.NoError(t, err)
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFundingRateHistory(contextGenerate(), "BTC-USD-SWAP", time.Time{}, time.Time{}, 2)
	assert.NoError(t, err)
}

func TestGetLimitPrice(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLimitPrice(contextGenerate(), "BTC-USD-SWAP")
	assert.NoError(t, err)
}

func TestGetOptionMarketData(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOptionMarketData(contextGenerate(), "BTC-USD", time.Time{})
	assert.NoError(t, err)
}

func TestGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	r, err := ok.FetchTradablePairs(contextGenerate(), asset.Futures)
	assert.NoError(t, err)
	_, err = ok.GetEstimatedDeliveryPrice(contextGenerate(), r[0].String())
	assert.NoError(t, err)
}

func TestGetDiscountRateAndInterestFreeQuota(t *testing.T) {
	t.Parallel()
	_, err := ok.GetDiscountRateAndInterestFreeQuota(contextGenerate(), "", 0)
	assert.NoError(t, err, "Okx GetDiscountRateAndInterestFreeQuota() error", err)
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSystemTime(contextGenerate())
	assert.NoError(t, err)
}

func TestGetLiquidationOrders(t *testing.T) {
	t.Parallel()
	insts, err := ok.FetchTradablePairs(contextGenerate(), asset.Margin)
	if err != nil {
		t.Skip(err)
	}
	_, err = ok.GetLiquidationOrders(contextGenerate(), &LiquidationOrderRequestParams{
		InstrumentType: okxInstTypeMargin,
		Underlying:     insts[0].String(),
		Currency:       currency.BTC,
		Limit:          2,
	})
	assert.NoError(t, err)
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := ok.GetMarkPrice(contextGenerate(), "MARGIN", "", "")
	assert.NoError(t, err)
}

func TestGetPositionTiers(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPositionTiers(contextGenerate(), "FUTURES", "cross", "BTC-USDT", "", "")
	assert.NoError(t, err)
}

func TestGetInterestRateAndLoanQuota(t *testing.T) {
	t.Parallel()
	_, err := ok.GetInterestRateAndLoanQuota(contextGenerate())
	assert.NoError(t, err)
}

func TestGetInterestRateAndLoanQuotaForVIPLoans(t *testing.T) {
	t.Parallel()
	_, err := ok.GetInterestRateAndLoanQuotaForVIPLoans(contextGenerate())
	assert.NoError(t, err)
}

func TestGetPublicUnderlyings(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicUnderlyings(contextGenerate(), "swap")
	assert.NoError(t, err)
}

func TestGetInsuranceFundInformation(t *testing.T) {
	t.Parallel()
	r, err := ok.GetInsuranceFundInformation(contextGenerate(), &InsuranceFundInformationRequestParams{
		InstrumentType: "FUTURES",
		Underlying:     "BTC-USDT",
		Limit:          2,
	})
	assert.NoError(t, err, "GetInsuranceFundInformation should not error")
	assert.Positive(t, r.Total, "Total should be positive")
	assert.NotEmpty(t, r.Details, "Should have some details")
	for _, d := range r.Details {
		assert.Positive(t, d.Balance, "Balance should be positive")
		assert.NotEmpty(t, d.Type, "Type should not be empty")
		assert.Positive(t, d.Timestamp, "Timestamp should be positive")
	}
}

func TestCurrencyUnitConvert(t *testing.T) {
	t.Parallel()
	_, err := ok.CurrencyUnitConvert(contextGenerate(), "BTC-USD-SWAP", 1, 3500, 1, "")
	assert.NoError(t, err)
}

// Trading related endpoints test functions.
func TestGetSupportCoins(t *testing.T) {
	t.Parallel()
	_, err := ok.GetSupportCoins(contextGenerate())
	assert.NoError(t, err)
}

func TestGetTakerVolume(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTakerVolume(contextGenerate(), "BTC", "SPOT", time.Time{}, time.Time{}, kline.OneDay)
	assert.NoError(t, err)
}
func TestGetMarginLendingRatio(t *testing.T) {
	t.Parallel()
	_, err := ok.GetMarginLendingRatio(contextGenerate(), "BTC", time.Time{}, time.Time{}, kline.FiveMin)
	assert.NoError(t, err)
}

func TestGetLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLongShortRatio(contextGenerate(), "BTC", time.Time{}, time.Time{}, kline.OneDay)
	assert.NoError(t, err)
}

func TestGetContractsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	_, err := ok.GetContractsOpenInterestAndVolume(contextGenerate(), "BTC", time.Time{}, time.Time{}, kline.OneDay)
	assert.NoError(t, err)
}

func TestGetOptionsOpenInterestAndVolume(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOptionsOpenInterestAndVolume(contextGenerate(), "BTC", kline.OneDay)
	assert.NoError(t, err)
}

func TestGetPutCallRatio(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPutCallRatio(contextGenerate(), "BTC", kline.OneDay)
	assert.NoError(t, err)
}

func TestGetOpenInterestAndVolumeExpiry(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOpenInterestAndVolumeExpiry(contextGenerate(), "BTC", kline.OneDay)
	assert.NoError(t, err)
}

func TestGetOpenInterestAndVolumeStrike(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOpenInterestAndVolumeStrike(contextGenerate(), "BTC", time.Now(), kline.OneDay)
	assert.NoError(t, err)
}

func TestGetTakerFlow(t *testing.T) {
	t.Parallel()
	_, err := ok.GetTakerFlow(contextGenerate(), "BTC", kline.OneDay)
	assert.NoError(t, err)
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.PlaceOrder(contextGenerate(), &PlaceOrderRequestParam{
		InstrumentID: "BTC-USDC",
		TradeMode:    "cross",
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       2.6,
		Price:        2.1,
		Currency:     "BTC",
	}, asset.Margin)
	assert.NoError(t, err)
}

const (
	instrumentJSON                                = `{"alias":"","baseCcy":"","category":"1","ctMult":"1","ctType":"linear","ctVal":"0.0001","ctValCcy":"BTC","expTime":"","instFamily":"BTC-USDC","instId":"BTC-USDC-SWAP","instType":"SWAP","lever":"125","listTime":"1666076190000","lotSz":"1","maxIcebergSz":"100000000.0000000000000000","maxLmtSz":"100000000","maxMktSz":"85000","maxStopSz":"85000","maxTriggerSz":"100000000.0000000000000000","maxTwapSz":"","minSz":"1","optType":"","quoteCcy":"","settleCcy":"USDC","state":"live","stk":"","tickSz":"0.1","uly":"BTC-USDC"}`
	placeOrderArgs                                = `[{"side": "buy","instId": "BTC-USDT","tdMode": "cash","ordType": "market","sz": "100"},{"side": "buy","instId": "LTC-USDT","tdMode": "cash","ordType": "market","sz": "1"}]`
	calculateOrderbookChecksumUpdateorderbookJSON = `{"Bids":[{"Amount":56,"Price":0.07014,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":608,"Price":0.07011,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":110,"Price":0.07009,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1264,"Price":0.07006,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":2347,"Price":0.07004,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":279,"Price":0.07003,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":52,"Price":0.07001,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":91,"Price":0.06997,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4242,"Price":0.06996,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":486,"Price":0.06995,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":161,"Price":0.06992,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":63,"Price":0.06991,"ID":0,"Period":0,"LiquidationOrders":0,
	"OrderCount":0},{"Amount":7518,"Price":0.06988,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":186,"Price":0.06976,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":71,"Price":0.06975,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1086,"Price":0.06973,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":513,"Price":0.06961,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4603,"Price":0.06959,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":186,"Price":0.0695,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3043,"Price":0.06946,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":103,"Price":0.06939,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5053,"Price":0.0693,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5039,"Price":0.06909,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5037,"Price":0.06888,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1526,"Price":0.06886,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5008,"Price":0.06867,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5065,"Price":0.06846,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1572,"Price":0.06826,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1565,"Price":0.06801,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":67,"Price":0.06748,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":111,"Price":0.0674,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10038,"Price":0.0672,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.06652,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1526,"Price":0.06625,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10924,"Price":0.06619,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.05986,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.05387,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.04848,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.04363,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0}],"Asks":[{"Amount":5,"Price":0.07026,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":765,"Price":0.07027,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":110,"Price":0.07028,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1264,"Price":0.0703,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":280,"Price":0.07034,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":2255,"Price":0.07035,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":28,"Price":0.07036,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":63,"Price":0.07037,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":137,"Price":0.07039,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":48,"Price":0.0704,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":32,"Price":0.07041,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3985,"Price":0.07043,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":257,"Price":0.07057,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":7870,"Price":0.07058,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":161,"Price":0.07059,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":4539,"Price":0.07061,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1438,"Price":0.07068,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":3162,"Price":0.07088,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":99,"Price":0.07104,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5018,"Price":0.07108,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1540,"Price":0.07115,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5080,"Price":0.07129,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1512,"Price":0.07145,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5016,"Price":0.0715,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5026,"Price":0.07171,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":5062,"Price":0.07192,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1517,"Price":0.07197,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1511,"Price":0.0726,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10376,"Price":0.07314,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.07354,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":10277,"Price":0.07466,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":269,"Price":0.07626,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":269,"Price":0.07636,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.0809,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.08899,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.09789,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0},{"Amount":1,"Price":0.10768,"ID":0,"Period":0,"LiquidationOrders":0,"OrderCount":0}],"Exchange":"Okx","Pair":"BTC-USDT","Asset":"spot","LastUpdated":"0001-01-01T00:00:00Z","LastUpdateID":0,"PriceDuplication":false,"IsFundingRate":false,"RestSnapshot":false,"IDAlignment":false}`
	placeMultipleOrderParamsJSON = `[{"instId":"BTC-USDT","tdMode":"cash","clOrdId":"b159","side":"buy","ordType":"limit","px":"2.15","sz":"2"},{"instId":"BTC-USDT","tdMode":"cash","clOrdId":"b15","side":"buy","ordType":"limit","px":"2.15","sz":"2"}]`
)

func TestPlaceMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var params []PlaceOrderRequestParam
	err := json.Unmarshal([]byte(placeMultipleOrderParamsJSON), &params)
	assert.NoError(t, err, err)

	_, err = ok.PlaceMultipleOrders(contextGenerate(),
		params)
	assert.NoError(t, err)
}

func TestCancelSingleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.CancelSingleOrder(contextGenerate(),
		CancelOrderRequestParam{
			InstrumentID: "BTC-USDT",
			OrderID:      "2510789768709120",
		})
	assert.NoError(t, err)
}

func TestCancelMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelMultipleOrders(contextGenerate(), []CancelOrderRequestParam{{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
	}})
	assert.NoError(t, err)
}

func TestAmendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.AmendOrder(contextGenerate(), &AmendOrderRequestParams{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	})
	assert.NoError(t, err)
}
func TestAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.AmendMultipleOrders(contextGenerate(), []AmendOrderRequestParams{{
		InstrumentID: "BTC-USDT",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
	}})
	assert.NoError(t, err)
}

func TestClosePositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.ClosePositions(contextGenerate(), &ClosePositionsRequestParams{
		InstrumentID: "BTC-USDT",
		MarginMode:   "cross",
		Currency:     "BTC",
	})
	assert.NoError(t, err)
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetOrderDetail(contextGenerate(), &OrderDetailRequestParam{
		InstrumentID: "BTC-USDT",
		OrderID:      "2510789768709120",
	})
	assert.False(t, !strings.Contains(err.Error(), "Order does not exist"), err)
}

func TestGetOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetOrderList(contextGenerate(), &OrderListRequestParams{
		Limit: 1,
	})
	assert.NoError(t, err)
}

func TestGet7And3MonthDayOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.Get7DayOrderHistory(contextGenerate(), &OrderHistoryRequestParams{
		OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"},
	})
	assert.NoError(t, err)
	_, err = ok.Get3MonthOrderHistory(contextGenerate(), &OrderHistoryRequestParams{
		OrderListRequestParams: OrderListRequestParams{InstrumentType: "MARGIN"},
	})
	assert.NoError(t, err)
}

func TestTransactionHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetTransactionDetailsLast3Days(contextGenerate(), &TransactionDetailRequestParams{
		InstrumentType: "MARGIN",
		Limit:          1,
	})
	assert.NoError(t, err)
	_, err = ok.GetTransactionDetailsLast3Months(contextGenerate(), &TransactionDetailRequestParams{
		InstrumentType: "MARGIN",
	})
	assert.NoError(t, err)
}

func TestSetTransactionDetailIntervalFor2Years(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.SetTransactionDetailIntervalFor2Years(context.Background(), &FillArchiveParam{
		Year:    2022,
		Quarter: "Q2",
	})
	assert.NoError(t, err)
}

func TestStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.PlaceStopOrder(contextGenerate(), &AlgoOrderParams{
		TakeProfitTriggerPriceType: "index",
		InstrumentID:               "BTC-USDT",
		OrderType:                  "conditional",
		Side:                       order.Sell,
		TradeMode:                  "isolated",
		Size:                       12,

		TakeProfitTriggerPrice: 12335,
		TakeProfitOrderPrice:   1234,
	})
	assert.NoError(t, err)
	_, err = ok.PlaceTrailingStopOrder(contextGenerate(), &AlgoOrderParams{
		CallbackRatio: 0.01,
		InstrumentID:  "BTC-USDT",
		OrderType:     "move_order_stop",
		Side:          order.Buy,
		TradeMode:     "isolated",
		Size:          2,
		ActivePrice:   1234,
	})
	assert.NoError(t, err)
	_, err = ok.PlaceIcebergOrder(contextGenerate(), &AlgoOrderParams{
		PriceLimit:  100.22,
		SizeLimit:   9999.9,
		PriceSpread: "0.04",

		InstrumentID: "BTC-USDT",
		OrderType:    "iceberg",
		Side:         order.Buy,

		TradeMode: "isolated",
		Size:      6,
	})
	assert.NoError(t, err)
	_, err = ok.PlaceTWAPOrder(contextGenerate(), &AlgoOrderParams{
		InstrumentID: "BTC-USDT",
		PriceLimit:   100.22,
		SizeLimit:    9999.9,
		OrderType:    "twap",
		PriceSpread:  "0.4",
		TradeMode:    "cross",
		Side:         order.Sell,
		Size:         6,
		TimeInterval: kline.ThreeDay,
	})
	assert.NoError(t, err)
	_, err = ok.TriggerAlgoOrder(contextGenerate(), &AlgoOrderParams{
		TriggerPriceType: "mark",
		TriggerPrice:     1234,

		InstrumentID: "BTC-USDT",
		OrderType:    "trigger",
		Side:         order.Buy,
		TradeMode:    "cross",
		Size:         5,
	})
	assert.NoError(t, err)
}

func TestCancelAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{
		{
			InstrumentID: "BTC-USDT",
			AlgoOrderID:  "90994943",
		},
	})
	assert.NoError(t, err)
}

func TestCancelAdvanceAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelAdvanceAlgoOrder(contextGenerate(), []AlgoOrderCancelParams{{
		InstrumentID: "BTC-USDT",
		AlgoOrderID:  "90994943",
	}})
	assert.NoError(t, err)
}

func TestGetAlgoOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetAlgoOrderList(contextGenerate(), "conditional", "", "", "", "", time.Time{}, time.Time{}, 1)
	assert.NoError(t, err)
}

func TestGetAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetAlgoOrderHistory(contextGenerate(), "conditional", "effective", "", "", "", time.Time{}, time.Time{}, 1)
	assert.NoError(t, err)
}

func TestGetEasyConvertCurrencyList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetEasyConvertCurrencyList(contextGenerate())
	assert.NoError(t, err)
}

func TestGetOneClickRepayCurrencyList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetOneClickRepayCurrencyList(contextGenerate(), "cross")
	assert.False(t, err != nil && !strings.Contains(err.Error(), "Parameter acctLv  error"), err)
}

func TestPlaceEasyConvert(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.PlaceEasyConvert(contextGenerate(),
		PlaceEasyConvertParam{
			FromCurrency: []string{"BTC"},
			ToCurrency:   "USDT"})
	assert.NoError(t, err, "%s PlaceEasyConvert() error %v", ok.Name, err)
}

func TestGetEasyConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetEasyConvertHistory(contextGenerate(), time.Time{}, time.Time{}, 1)
	assert.NoError(t, err)
}

func TestGetOneClickRepayHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetOneClickRepayHistory(contextGenerate(), time.Time{}, time.Time{}, 1)
	assert.Falsef(t, err != nil && !strings.Contains(err.Error(), "Parameter acctLv  error"), "%s GetOneClickRepayHistory() error %v", ok.Name, err)
}

func TestTradeOneClickRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.TradeOneClickRepay(contextGenerate(), TradeOneClickRepayParam{
		DebtCurrency:  []string{"BTC"},
		RepayCurrency: "USDT",
	})
	assert.NoError(t, err)
}

func TestGetCounterparties(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetCounterparties(contextGenerate()); err != nil && !strings.Contains(err.Error(), "code: 70006 message: Does not meet the minimum asset requirement.") {
		t.Error("Okx GetCounterparties() error", err)
	}
}

const createRfqInputJSON = `{"anonymous": true,"counterparties":["Trader1","Trader2"],"clRfqId":"rfq01","legs":[{"sz":"25","side":"buy","instId":"BTCUSD-221208-100000-C"},{"sz":"150","side":"buy","instId":"ETH-USDT","tgtCcy":"base_ccy"}]}`

func TestCreateRfq(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var input CreateRfqInput
	err := json.Unmarshal([]byte(createRfqInputJSON), &input)
	assert.NoError(t, err)
	_, err = ok.CreateRfq(contextGenerate(), input)
	assert.NoError(t, err)
}

func TestCancelRfq(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelRfq(contextGenerate(), CancelRfqRequestParam{})
	if err != nil && !errors.Is(err, errMissingRfqIDAndClientRfqID) {
		t.Errorf("Okx CancelRfq() expecting %v, but found %v", errMissingRfqIDAndClientRfqID, err)
	}
	_, err = ok.CancelRfq(context.Background(), CancelRfqRequestParam{
		ClientRfqID: "somersdjskfjsdkfjxvxv",
	})
	assert.NoError(t, err, "Okx CancelRfq() error", err)
}

func TestMultipleCancelRfq(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelMultipleRfqs(contextGenerate(), CancelRfqRequestsParam{})
	if err != nil && !errors.Is(err, errMissingRfqIDAndClientRfqID) {
		t.Errorf("Okx CancelMultipleRfqs() expecting %v, but found %v", errMissingRfqIDAndClientRfqID, err)
	}
	_, err = ok.CancelMultipleRfqs(contextGenerate(), CancelRfqRequestsParam{
		ClientRfqIDs: []string{"somersdjskfjsdkfjxvxv"},
	})
	assert.NoError(t, err, "Okx CancelMultipleRfqs() error", err)
}

func TestCancelAllRfqs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.CancelAllRfqs(contextGenerate())
	assert.NoError(t, err)
}

func TestExecuteQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.ExecuteQuote(contextGenerate(), ExecuteQuoteParams{})
	assert.Falsef(t, err != nil && !errors.Is(err, errMissingRfqIDOrQuoteID), "Okx ExecuteQuote() expected %v, but found %v", errMissingRfqIDOrQuoteID, err)
	_, err = ok.ExecuteQuote(contextGenerate(), ExecuteQuoteParams{
		RfqID:   "22540",
		QuoteID: "84073",
	})
	assert.NoError(t, err)
}

func TestGetQuoteProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetQuoteProducts(context.Background())
	assert.NoError(t, err)
}

func TestSetQuoteProducts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.SetQuoteProducts(contextGenerate(), []SetQuoteProductParam{
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
		}})
	assert.NoError(t, err)
}

func TestResetRFQMMPStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.ResetRFQMMPStatus(context.Background())
	assert.NoError(t, err)
}

func TestCreateQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.CreateQuote(contextGenerate(), CreateQuoteParams{}); err != nil && !errors.Is(err, errMissingRfqID) {
		t.Errorf("Okx CreateQuote() expecting %v, but found %v", errMissingRfqID, err)
	}
	_, err := ok.CreateQuote(contextGenerate(), CreateQuoteParams{
		RfqID:     "12345",
		QuoteSide: order.Buy.Lower(),
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
	})
	assert.NoError(t, err)
}

func TestCancelQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.CancelQuote(contextGenerate(), nil)
	assert.Falsef(t, err != nil && !errors.Is(err, errNilArgument), "expected %v, got %v", errNilArgument, err)
	_, err = ok.CancelQuote(contextGenerate(), &CancelQuoteRequestParams{})
	assert.False(t, err != nil && !errors.Is(err, errMissingQuoteIDOrClientQuoteID), err)
	_, err = ok.CancelQuote(contextGenerate(), &CancelQuoteRequestParams{
		QuoteID: "1234",
	})
	assert.NoError(t, err)
	_, err = ok.CancelQuote(contextGenerate(), &CancelQuoteRequestParams{
		ClientQuoteID: "1234",
	})
	assert.NoError(t, err)
}

func TestCancelMultipleQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelMultipleQuote(contextGenerate(), CancelQuotesRequestParams{})
	assert.Falsef(t, err != nil && !errors.Is(errMissingEitherQuoteIDAOrClientQuoteIDs, err), "Okx CancelQuote() error", err)
	_, err = ok.CancelMultipleQuote(contextGenerate(), CancelQuotesRequestParams{
		QuoteIDs: []string{"1150", "1151", "1152"},
		// Block trades require a minimum of $100,000 in assets in your trading account
	})
	assert.NoError(t, err)
}

func TestCancelAllQuotes(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	time, err := ok.CancelAllQuotes(contextGenerate())
	switch {
	case err != nil:
		t.Error("Okx CancelAllQuotes() error", err)
	case err == nil && time.IsZero():
		t.Error("Okx CancelAllQuotes() zero timestamp message ")
	}
}

func TestGetRfqs(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetRfqs(contextGenerate(), &RfqRequestParams{
		Limit: 1,
	})
	assert.NoError(t, err)
}

func TestGetQuotes(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetQuotes(contextGenerate(), &QuoteRequestParams{
		Limit: 3,
	})
	assert.NoError(t, err)
}

func TestGetRfqTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetRfqTrades(contextGenerate(), &RfqTradesRequestParams{
		Limit: 1,
	})
	assert.NoError(t, err)
}

func TestGetPublicTrades(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetPublicTrades(contextGenerate(), "", "", 3)
	assert.NoError(t, err)
}

func TestGetFundingCurrencies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetFundingCurrencies(contextGenerate(), "BTC")
	assert.NoError(t, err)
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetBalance(contextGenerate(), "")
	assert.NoError(t, err)
}

func TestGetNonTradableAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetNonTradableAssets(context.Background(), "BTC")
	assert.NoError(t, err)
}

func TestGetAccountAssetValuation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetAccountAssetValuation(contextGenerate(), "")
	assert.NoError(t, err)
}

func TestFundingTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.FundingTransfer(contextGenerate(), &FundingTransferRequestInput{
		Amount:   12.000,
		To:       "6",
		From:     "18",
		Currency: "BTC",
	})
	assert.NoError(t, err)
}

func TestGetFundsTransferState(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetFundsTransferState(contextGenerate(), "754147", "1232", 1)
	assert.False(t, err != nil && !strings.Contains(err.Error(), "Parameter transId  error"), err)
}

func TestGetAssetBillsDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetAssetBillsDetails(contextGenerate(), "", "", time.Time{}, time.Time{}, 0, 1)
	assert.NoError(t, err)
}

func TestGetLightningDeposits(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetLightningDeposits(contextGenerate(), "BTC", 1.00, 0); err != nil && !strings.Contains(err.Error(), "58355") {
		t.Error("Okx GetLightningDeposits() error", err)
	}
}

func TestGetCurrencyDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetCurrencyDepositAddress(contextGenerate(), "BTC")
	assert.NoError(t, err)
}

func TestGetCurrencyDepositHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetCurrencyDepositHistory(contextGenerate(), "BTC", "", "", time.Time{}, time.Time{}, 0, 1)
	assert.NoError(t, err)
}

func TestWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.Withdrawal(contextGenerate(), &WithdrawalInput{Amount: 0.1, TransactionFee: 0.00005, Currency: "BTC", WithdrawalDestination: "4", ToAddress: core.BitcoinDonationAddress})
	assert.NoError(t, err, "Okx Withdrawal error", err)
}

func TestLightningWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.LightningWithdrawal(contextGenerate(), LightningWithdrawalRequestInput{
		Currency: currency.BTC.String(),
		Invoice:  "lnbc100u1psnnvhtpp5yq2x3q5hhrzsuxpwx7ptphwzc4k4wk0j3stp0099968m44cyjg9sdqqcqzpgxqzjcsp5hz",
	})
	assert.NoError(t, err)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelWithdrawal(contextGenerate(), "fjasdfkjasdk")
	assert.NoError(t, err)
}

func TestGetWithdrawalHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetWithdrawalHistory(contextGenerate(), "BTC", "", "", "", "", time.Time{}, time.Time{}, 1)
	assert.NoError(t, err)
}

func TestSmallAssetsConvert(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.SmallAssetsConvert(contextGenerate(), []string{"BTC", "USDT"})
	assert.NoError(t, err)
}

func TestGetSavingBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetSavingBalance(contextGenerate(), "BTC")
	assert.NoError(t, err)
}

func TestSavingsPurchase(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.SavingsPurchaseOrRedemption(contextGenerate(), &SavingsPurchaseRedemptionInput{
		Amount:     123.4,
		Currency:   "BTC",
		Rate:       1,
		ActionType: "purchase",
	})
	assert.NoError(t, err)
	_, err = ok.SavingsPurchaseOrRedemption(contextGenerate(), &SavingsPurchaseRedemptionInput{
		Amount:     123.4,
		Currency:   "BTC",
		Rate:       1,
		ActionType: "redempt",
	})
	assert.NoError(t, err)
}

func TestSetLendingRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.SetLendingRate(contextGenerate(), LendingRate{Currency: "BTC", Rate: 2})
	assert.NoError(t, err)
}

func TestGetLendingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetLendingHistory(contextGenerate(), "USDT", time.Time{}, time.Time{}, 1)
	assert.NoError(t, err)
}

func TestGetPublicBorrowInfo(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicBorrowInfo(contextGenerate(), "")
	assert.NoError(t, err)
	_, err = ok.GetPublicBorrowInfo(context.Background(), "USDT")
	assert.NoError(t, err)
}

func TestGetPublicBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicBorrowHistory(context.Background(), "USDT", time.Time{}, time.Time{}, 1)
	assert.NoError(t, err)
}

func TestGetConvertCurrencies(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetConvertCurrencies(contextGenerate())
	assert.NoError(t, err)
}

func TestGetConvertCurrencyPair(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetConvertCurrencyPair(contextGenerate(), "USDT", "BTC")
	assert.NoError(t, err)
}

func TestEstimateQuote(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.EstimateQuote(contextGenerate(), &EstimateQuoteRequestInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          "sell",
		RfqAmount:     30,
		RfqSzCurrency: "USDT",
	})
	assert.NoError(t, err)
}

func TestConvertTrade(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.ConvertTrade(contextGenerate(), &ConvertTradeInput{
		BaseCurrency:  "BTC",
		QuoteCurrency: "USDT",
		Side:          "Buy",
		Size:          2,
		SizeCurrency:  "USDT",
		QuoteID:       "quoterETH-USDT16461885104612381",
	})
	assert.NoError(t, err)
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetConvertHistory(contextGenerate(), time.Time{}, time.Time{}, 1, "")
	assert.NoError(t, err)
}

func TestGetNonZeroAccountBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.AccountBalance(contextGenerate(), "")
	assert.NoError(t, err)
}

func TestGetPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetPositions(contextGenerate(), "", "", "")
	assert.NoError(t, err)
}

func TestGetPositionsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetPositionsHistory(contextGenerate(), "", "", "", 0, 1, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetAccountAndPositionRisk(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetAccountAndPositionRisk(contextGenerate(), "")
	assert.NoError(t, err)
}

func TestGetBillsDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetBillsDetailLast7Days(contextGenerate(), &BillsDetailQueryParameter{
		Limit: 3,
	})
	assert.NoError(t, err)
}

func TestGetAccountConfiguration(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetAccountConfiguration(contextGenerate())
	assert.NoError(t, err)
}

func TestSetPositionMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.SetPositionMode(contextGenerate(), "net_mode")
	assert.NoError(t, err)
}

func TestSetLeverageRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.SetLeverageRate(contextGenerate(), SetLeverageInput{
		Currency:     "USDT",
		Leverage:     5,
		MarginMode:   "cross",
		InstrumentID: "BTC-USDT",
	}); err != nil && !errors.Is(err, errNoValidResponseFromServer) {
		t.Error("Okx SetLeverageRate() error", err)
	}
}

func TestGetMaximumBuySellAmountOROpenAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetMaximumBuySellAmountOROpenAmount(contextGenerate(), "BTC-USDT", "cross", "BTC", "", 5)
	assert.NoError(t, err)
}

func TestGetMaximumAvailableTradableAmount(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetMaximumAvailableTradableAmount(contextGenerate(), "BTC-USDT", "BTC", "cross", true, 123)
	assert.False(t, err != nil && !strings.Contains(err.Error(), "51010"), err)
}

func TestIncreaseDecreaseMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.IncreaseDecreaseMargin(contextGenerate(), IncreaseDecreaseMarginInput{
		InstrumentID: "BTC-USDT",
		PositionSide: "long",
		Type:         "add",
		Amount:       1000,
		Currency:     "USD",
	})
	assert.NoError(t, err)
}

func TestGetLeverageRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetLeverageRate(contextGenerate(), "BTC-USDT", "cross")
	assert.NoError(t, err)
}

func TestGetMaximumLoanOfInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetMaximumLoanOfInstrument(contextGenerate(), "ZRX-BTC", "isolated", "ZRX")
	assert.False(t, err != nil && !strings.Contains(err.Error(), "51010"), err)
}

func TestGetTradeFee(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetTradeFee(contextGenerate(), "SPOT", "", "")
	assert.NoError(t, err)
}

func TestGetInterestAccruedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetInterestAccruedData(contextGenerate(), 0, 1, "", "", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetInterestRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetInterestRate(contextGenerate(), "")
	assert.NoError(t, err)
}

func TestSetGreeks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.SetGreeks(contextGenerate(), "PA")
	assert.NoError(t, err)
}

func TestIsolatedMarginTradingSettings(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.IsolatedMarginTradingSettings(contextGenerate(), IsolatedMode{
		IsoMode:        "autonomy",
		InstrumentType: "MARGIN",
	})
	assert.NoError(t, err)
}

func TestGetMaximumWithdrawals(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetMaximumWithdrawals(contextGenerate(), "BTC")
	assert.NoError(t, err)
}

func TestGetAccountRiskState(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetAccountRiskState(contextGenerate())
	assert.False(t, err != nil && !strings.Contains(err.Error(), "51010"), err)
}

func TestVIPLoansBorrowAndRepay(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.VIPLoansBorrowAndRepay(contextGenerate(), LoanBorrowAndReplayInput{Currency: "BTC", Side: "borrow", Amount: 12})
	assert.NoError(t, err)
}

func TestGetBorrowAndRepayHistoryForVIPLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetBorrowAndRepayHistoryForVIPLoans(contextGenerate(), "", time.Time{}, time.Time{}, 3)
	assert.NoError(t, err)
}

func TestGetBorrowInterestAndLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetBorrowInterestAndLimit(contextGenerate(), 1, "BTC")
	assert.NoError(t, err)
}

func TestPositionBuilder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.PositionBuilder(contextGenerate(), PositionBuilderInput{
		ImportExistingPosition: true,
	})
	assert.NoError(t, err)
}

func TestGetGreeks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetGreeks(contextGenerate(), "")
	assert.False(t, err != nil && !strings.Contains(err.Error(), "Unsupported operation"), err)
}

func TestGetPMLimitation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetPMPositionLimitation(contextGenerate(), "SWAP", "BTC-USDT")
	assert.NoError(t, err)
}

func TestViewSubaccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.ViewSubAccountList(contextGenerate(), false, "", time.Time{}, time.Time{}, 2)
	assert.NoError(t, err)
}

func TestResetSubAccountAPIKey(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{
		SubAccountName:   "sam",
		APIKey:           apiKey,
		APIKeyPermission: "trade",
	}); err != nil && !strings.Contains(err.Error(), "Parameter subAcct can not be empty.") {
		t.Errorf("%s ResetSubAccountAPIKey() error %v", ok.Name, err)
	}
	if _, err := ok.ResetSubAccountAPIKey(contextGenerate(), &SubAccountAPIKeyParam{
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

	if _, err := ok.GetSubaccountTradingBalance(contextGenerate(), ""); err != nil && !errors.Is(err, errMissingRequiredParameterSubaccountName) {
		t.Errorf("Okx GetSubaccountTradingBalance() expecting \"%v\", but found \"%v\"", errMissingRequiredParameterSubaccountName, err)
	}
	if _, err := ok.GetSubaccountTradingBalance(contextGenerate(), "test1"); err != nil && !strings.Contains(err.Error(), "sub-account does not exist") {
		t.Error("Okx GetSubaccountTradingBalance() error", err)
	}
}

func TestGetSubaccountFundingBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetSubaccountFundingBalance(contextGenerate(), "test1", ""); err != nil && !strings.Contains(err.Error(), "Sub-account test1 does not exists") && !strings.Contains(err.Error(), "59510") {
		t.Error("Okx GetSubaccountFundingBalance() error", err)
	}
}
func TestGetSubAccountMaximumWithdrawal(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetSubAccountMaximumWithdrawal(context.Background(), "test1", "BTC")
	assert.NoError(t, err, "Okx GetSubaccountFundingBalance() error", err)
}

func TestHistoryOfSubaccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.HistoryOfSubaccountTransfer(contextGenerate(), "", "0", "", time.Time{}, time.Time{}, 1)
	assert.NoError(t, err)
}

func TestGetHistoryOfManagedSubAccountTransfer(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetHistoryOfManagedSubAccountTransfer(context.Background(), "BTC", "", "", "", time.Time{}, time.Time{}, 10)
	assert.NoError(t, err)
}

func TestMasterAccountsManageTransfersBetweenSubaccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), SubAccountAssetTransferParams{Currency: "BTC", Amount: 1200, From: 9, To: 9, FromSubAccount: "", ToSubAccount: "", LoanTransfer: true}); err != nil && !errors.Is(err, errInvalidSubaccount) {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", err)
	}
	if _, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), SubAccountAssetTransferParams{Currency: "BTC", Amount: 1200, From: 8, To: 8, FromSubAccount: "", ToSubAccount: "", LoanTransfer: true}); err != nil && !errors.Is(err, errInvalidSubaccount) {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", err)
	}
	if _, err := ok.MasterAccountsManageTransfersBetweenSubaccounts(contextGenerate(), SubAccountAssetTransferParams{Currency: "BTC", Amount: 1200, From: 6, To: 6, FromSubAccount: "test1", ToSubAccount: "test2", LoanTransfer: true}); err != nil && !strings.Contains(err.Error(), "Sub-account test1 does not exists") {
		t.Error("Okx MasterAccountsManageTransfersBetweenSubaccounts() error", err)
	}
}

func TestSetPermissionOfTransferOut(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.SetPermissionOfTransferOut(contextGenerate(), PermissionOfTransfer{SubAcct: "Test1"}); err != nil && !strings.Contains(err.Error(), "Sub-account does not exist") {
		t.Error("Okx SetPermissionOfTransferOut() error", err)
	}
}

func TestGetCustodyTradingSubaccountList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetCustodyTradingSubaccountList(contextGenerate(), "")
	assert.NoError(t, err)
}

func TestSetSubAccountVIPLoanAllocation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.SetSubAccountVIPLoanAllocation(context.Background(), &SubAccountLoanAllocationParam{
		Enable: true,
		Alloc: []subAccountVIPLoanAllocationInfo{
			{
				SubAcct:   "subAcct1",
				LoanAlloc: 20.01,
			},
		},
	})
	assert.NoError(t, err)
}

func TestGetSubAccountBorrowInterestAndLimit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetSubAccountBorrowInterestAndLimit(context.Background(), "123456", "ETH")
	assert.NoError(t, err)
}

// ETH Staking

func TestPurcahseETHStaking(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	err := ok.PurcahseETHStaking(context.Background(), 100)
	assert.NoError(t, err)
}

// RedeemETHStaking
func TestRedeemETHStaking(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	err := ok.RedeemETHStaking(context.Background(), 100)
	assert.NoError(t, err)
}

func TestGetBETHAssetsBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetBETHAssetsBalance(context.Background())
	assert.NoError(t, err)
}

func TestGetPurchaseAndRedeemHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetPurchaseAndRedeemHistory(context.Background(), "purchase", "pending", time.Time{}, time.Now(), 10)
	assert.NoError(t, err)
}

func TestGetAPYHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetAPYHistory(context.Background(), 34)
	assert.NoError(t, err)
}

const gridTradingPlaceOrder = `{"instId": "BTC-USD-SWAP","algoOrdType": "contract_grid","maxPx": "5000","minPx": "400","gridNum": "10","runType": "1","sz": "200", "direction": "long","lever": "2"}`

func TestPlaceGridAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var input GridAlgoOrder
	err := json.Unmarshal([]byte(gridTradingPlaceOrder), &input)
	assert.NoError(t, err)
	_, err = ok.PlaceGridAlgoOrder(contextGenerate(), &input)
	assert.NoError(t, err)
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
	err := json.Unmarshal([]byte(gridOrderAmendAlgo), &input)
	assert.NoError(t, err)
	_, err = ok.AmendGridAlgoOrder(contextGenerate(), input)
	assert.NoError(t, err)
}

const stopGridAlgoOrderJSON = `{"algoId":"198273485",	"instId":"BTC-USDT",	"stopType":"1",	"algoOrdType":"grid"}`

func TestStopGridAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var resp StopGridAlgoOrderRequest
	err := json.Unmarshal([]byte(stopGridAlgoOrderJSON), &resp)
	assert.NoError(t, err)
	if _, err := ok.StopGridAlgoOrder(contextGenerate(), []StopGridAlgoOrderRequest{
		resp,
	}); err != nil && !strings.Contains(err.Error(), "The strategy does not exist or has stopped") {
		t.Error("Okx StopGridAlgoOrder() error", err)
	}
}

func TestGetGridAlgoOrdersList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetGridAlgoOrdersList(contextGenerate(), "grid", "", "", "", "", "", 1)
	assert.NoError(t, err)
}

func TestGetGridAlgoOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetGridAlgoOrderHistory(contextGenerate(), "contract_grid", "", "", "", "", "", 1)
	assert.NoError(t, err)
}

func TestGetGridAlgoOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetGridAlgoOrderDetails(contextGenerate(), "grid", "")
	assert.Falsef(t, err != nil && !errors.Is(err, errMissingAlgoOrderID), "expecting %v, but found %v error", errMissingAlgoOrderID, err)
	_, err = ok.GetGridAlgoOrderDetails(contextGenerate(), "grid", "7878")
	assert.False(t, err != nil && !strings.Contains(err.Error(), "Order does not exist"), err)
}

func TestGetGridAlgoSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.GetGridAlgoSubOrders(contextGenerate(), "", "", "", "", "", "", 2); err != nil && !errors.Is(err, errMissingAlgoOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingAlgoOrderType)
	}
	if _, err := ok.GetGridAlgoSubOrders(contextGenerate(), "grid", "", "", "", "", "", 2); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingAlgoOrderID)
	}
	if _, err := ok.GetGridAlgoSubOrders(contextGenerate(), "grid", "1234", "", "", "", "", 2); err != nil && !errors.Is(err, errMissingSubOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingSubOrderType)
	}
	if _, err := ok.GetGridAlgoSubOrders(contextGenerate(), "grid", "1234", "live", "", "", "", 2); err != nil && !errors.Is(err, errMissingSubOrderType) {
		t.Errorf("Okx GetGridAlgoSubOrders() expecting %v, but found %v", err, errMissingSubOrderType)
	}
}

const spotGridAlgoOrderPosition = `{"adl": "1","algoId": "449327675342323712","avgPx": "29215.0142857142857149","cTime": "1653400065917","ccy": "USDT","imr": "2045.386","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29206.7","lever": "5","liqPx": "661.1684795867162","markPx": "29213.9","mgnMode": "cross","mgnRatio": "217.19370606167573","mmr": "40.907720000000005","notionalUsd": "10216.70307","pos": "35","posSide": "net","uTime": "1653400066938","upl": "1.674999999999818","uplRatio": "0.0008190504784478"}`

func TestGetGridAlgoOrderPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	var resp AlgoOrderPosition
	err := json.Unmarshal([]byte(spotGridAlgoOrderPosition), &resp)
	assert.NoError(t, err)
	if _, err := ok.GetGridAlgoOrderPositions(contextGenerate(), "", ""); err != nil && !errors.Is(err, errInvalidAlgoOrderType) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errInvalidAlgoOrderType, err)
	}
	if _, err := ok.GetGridAlgoOrderPositions(contextGenerate(), "contract_grid", ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
	if _, err := ok.GetGridAlgoOrderPositions(contextGenerate(), "contract_grid", ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
	if _, err := ok.GetGridAlgoOrderPositions(contextGenerate(), "contract_grid", "448965992920907776"); err != nil && !strings.Contains(err.Error(), "The strategy does not exist or has stopped") {
		t.Errorf("Okx GetGridAlgoOrderPositions() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
}

func TestSpotGridWithdrawProfit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.SpotGridWithdrawProfit(contextGenerate(), ""); err != nil && !errors.Is(err, errMissingAlgoOrderID) {
		t.Errorf("Okx SpotGridWithdrawProfit() expecting %v, but found %v", errMissingAlgoOrderID, err)
	}
	_, err := ok.SpotGridWithdrawProfit(contextGenerate(), "1234")
	assert.NoError(t, err)
}

func TestComputeMarginBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	if _, err := ok.ComputeMarginBalance(contextGenerate(), MarginBalanceParam{
		AlgoID: "123456",
		Type:   "other",
	}); err != nil && !errors.Is(err, errInvalidMarginTypeAdjust) {
		t.Errorf("%s ComputeMarginBalance() expected %v, but found %v", ok.Name, errInvalidMarginTypeAdjust, err)
	}
	if _, err := ok.ComputeMarginBalance(contextGenerate(), MarginBalanceParam{
		AlgoID: "123456",
		Type:   "add",
	}); err != nil && !strings.Contains(err.Error(), "The strategy does not exist or has stopped") {
		t.Errorf("%s ComputeMarginBalance() error %v", ok.Name, err)
	}
}

func TestAdjustMarginBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.AdjustMarginBalance(contextGenerate(), MarginBalanceParam{
		AlgoID: "1234",
		Type:   "add",
		Amount: 12345,
	})
	assert.NoError(t, err)
}

const gridAIParamJSON = `{"algoOrdType": "grid","annualizedRate": "1.5849","ccy": "USDT","direction": "",	"duration": "7D","gridNum": "5","instId": "BTC-USDT","lever": "0","maxPx": "21373.3","minInvestment": "0.89557758",	"minPx": "15544.2",	"perMaxProfitRate": "0.0733865364573281","perMinProfitRate": "0.0561101403446263","runType": "1"}`

func TestGetGridAIParameter(t *testing.T) {
	t.Parallel()
	var response GridAIParameterResponse
	err := json.Unmarshal([]byte(gridAIParamJSON), &response)
	assert.NoError(t, err)
	_, err = ok.GetGridAIParameter(contextGenerate(), "grid", "BTC-USDT", "", "")
	assert.NoError(t, err)
}
func TestGetOffers(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetOffers(contextGenerate(), "", "", "")
	assert.NoError(t, err)
}

func TestPurchase(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.Purchase(contextGenerate(), PurchaseRequestParam{
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
	})
	assert.NoError(t, err)
}

func TestRedeem(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	if _, err := ok.Redeem(contextGenerate(), RedeemRequestParam{
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

	if _, err := ok.CancelPurchaseOrRedemption(contextGenerate(), CancelFundingParam{
		OrderID:      "754147",
		ProtocolType: "defi",
	}); err != nil && !strings.Contains(err.Error(), "Order not found") {
		t.Errorf("%s CancelPurchaseOrRedemption() error %v", ok.Name, err)
	}
}

func TestGetEarnActiveOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetEarnActiveOrders(contextGenerate(), "", "", "", "")
	assert.NoError(t, err)
}

func TestGetFundingOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetFundingOrderHistory(contextGenerate(), "", "", "", time.Time{}, time.Time{}, 1)
	assert.NoError(t, err)
}

func TestSystemStatusResponse(t *testing.T) {
	t.Parallel()
	_, err := ok.SystemStatusResponse(contextGenerate(), "completed")
	assert.NoError(t, err)
}

/**********************************  Wrapper Functions **************************************/

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := ok.FetchTradablePairs(contextGenerate(), asset.Options)
	assert.NoError(t, err)
	_, err = ok.FetchTradablePairs(contextGenerate(), asset.PerpetualSwap)
	assert.NoError(t, err)
	_, err = ok.FetchTradablePairs(contextGenerate(), asset.Futures)
	assert.NoError(t, err)
	_, err = ok.FetchTradablePairs(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
	_, err = ok.FetchTradablePairs(contextGenerate(), asset.Spread)
	assert.NoError(t, err)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	err := ok.UpdateTradablePairs(contextGenerate(), true)
	assert.NoError(t, err)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	tests := map[asset.Item][]currency.Pair{
		asset.Spot: {
			currency.NewPair(currency.ETH, currency.USDT),
			currency.NewPair(currency.BTC, currency.USDT),
		},
		asset.Margin: {
			currency.NewPair(currency.ETH, currency.USDT),
			currency.NewPair(currency.ETH, currency.BTC),
		},
	}

	for _, a := range []asset.Item{asset.PerpetualSwap, asset.Futures, asset.Options} {
		pairs, err := ok.FetchTradablePairs(context.Background(), a)
		if assert.NoErrorf(t, err, "FetchTradablePairs should not error for %s", a) {
			tests[a] = []currency.Pair{pairs[0]}
		}
	}

	for _, a := range ok.GetAssetTypes(false) {
		if err := ok.UpdateOrderExecutionLimits(context.Background(), a); err != nil {
			t.Error("Okx UpdateOrderExecutionLimits() error", err)
			continue
		}

		for _, p := range tests[a] {
			limits, err := ok.GetOrderExecutionLimits(a, p)
			if assert.NoError(t, err, "GetOrderExecutionLimits should not error") {
				assert.Positivef(t, limits.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", p)
				assert.Positivef(t, limits.MinimumBaseAmount, "PriceStepIncrementSize should be positive for %s", p)
			}
		}
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.UpdateTicker(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	err := ok.UpdateTickers(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
	err = ok.UpdateTickers(contextGenerate(), asset.Spread)
	assert.NoError(t, err)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := ok.FetchTicker(contextGenerate(), currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")), asset.PerpetualSwap)
	assert.NoError(t, err)
	_, err = ok.FetchTicker(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	assert.NoError(t, err)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	_, err := ok.FetchOrderbook(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := ok.UpdateOrderbook(contextGenerate(), currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")), asset.Spot)
	assert.NoError(t, err)
	spreadPair, err := currency.NewPairDelimiter("BTC-USDT-SWAP_BTC-USDT-240126", currency.UnderscoreDelimiter)
	assert.NoError(t, err, err)
	_, err = ok.UpdateOrderbook(contextGenerate(), spreadPair, asset.Spread)
	assert.NoError(t, err)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.UpdateAccountInfo(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.FetchAccountInfo(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetAccountFundingHistory(contextGenerate())
	assert.NoError(t, err)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetWithdrawalsHistory(contextGenerate(), currency.BTC, asset.Spot)
	assert.NoError(t, err)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetRecentTrades(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.PerpetualSwap)
	assert.NoError(t, err)
	_, err = ok.GetRecentTrades(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spread)
	assert.NoError(t, err)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	var resp []PlaceOrderRequestParam
	err := json.Unmarshal([]byte(placeOrderArgs), &resp)
	assert.NoError(t, err, err)
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
	_, err = ok.SubmitOrder(contextGenerate(), orderSubmission)
	assert.NoError(t, err, "Okx SubmitOrder() error", err)

	cp, err := currency.NewPairFromString("BTC-USDT-230630")
	assert.NoError(t, err, err)

	orderSubmission = &order.Submit{
		Pair:       cp,
		Exchange:   ok.Name,
		Side:       order.Buy,
		Type:       order.Market,
		Amount:     1,
		ClientID:   "hellomoto",
		AssetType:  asset.Futures,
		MarginType: margin.Multi,
	}
	_, err = ok.SubmitOrder(contextGenerate(), orderSubmission)
	assert.NoError(t, err, "Okx SubmitOrder() error", err)
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
	err := ok.CancelOrder(contextGenerate(), orderCancellation)
	assert.NoError(t, err)
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
	_, err := ok.CancelBatchOrders(contextGenerate(), orderCancellationParams)
	if err != nil && !strings.Contains(err.Error(), "order does not exist.") {
		t.Error("Okx CancelBatchOrders() error", err)
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.CancelAllOrders(contextGenerate(), &order.Cancel{})
	assert.NoError(t, err)
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.ModifyOrder(contextGenerate(),
		&order.Modify{
			AssetType: asset.Spot,
			Pair:      currency.NewPair(currency.LTC, currency.BTC),
			OrderID:   "1234",
			Price:     123456.44,
			Amount:    123,
		})
	assert.NoErrorf(t, err, "Okx ModifyOrder() error %v", err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	enabled, err := ok.GetEnabledPairs(asset.Spot)
	assert.NoError(t, err, "couldn't find enabled tradable pairs")
	if len(enabled) == 0 {
		t.SkipNow()
	}
	_, err = ok.GetOrderInfo(contextGenerate(),
		"123", enabled[0], asset.Futures)
	assert.Truef(t, err == nil || strings.Contains(err.Error(), "Order does not exist"), "Okx GetOrderInfo() expecting %s, but found %v", "Order does not exist", err)
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetDepositAddress(contextGenerate(), currency.BTC, "", "")
	assert.False(t, err != nil && !errors.Is(err, errDepositAddressNotFound), err)
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
	_, err := ok.WithdrawCryptocurrencyFunds(contextGenerate(), &withdrawCryptoRequest)
	assert.NoError(t, err)
}

func TestGetPairFromInstrumentID(t *testing.T) {
	t.Parallel()
	instruments := []string{
		"BTC-USDT",
		"BTC-USDT-SWAP",
		"BTC-USDT-ER33234",
	}
	_, err := ok.GetPairFromInstrumentID(instruments[0])
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.Limit,
		Pairs:     currency.Pairs{pair, currency.NewPair(currency.USDT, currency.USD), currency.NewPair(currency.USD, currency.LTC)},
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	_, err = ok.GetActiveOrders(contextGenerate(), &getOrdersRequest)
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	var getOrdersRequest = order.MultiOrderRequest{
		Type:      order.AnyType,
		AssetType: asset.Spot,
		Side:      order.Buy,
	}
	_, err := ok.GetOrderHistory(contextGenerate(), &getOrdersRequest)
	if err == nil {
		t.Errorf("Okx GetOrderHistory() Expected: %v. received nil", err)
	} else if err != nil && !errors.Is(err, errMissingAtLeast1CurrencyPair) {
		t.Errorf("Okx GetOrderHistory() Expected: %v, but found %v", errMissingAtLeast1CurrencyPair, err)
	}
	getOrdersRequest.Pairs = []currency.Pair{
		currency.NewPair(currency.LTC,
			currency.BTC)}
	_, err = ok.GetOrderHistory(contextGenerate(), &getOrdersRequest)
	assert.NoError(t, err)
}
func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	_, err := ok.GetFeeByType(contextGenerate(), &exchange.FeeBuilder{
		Amount:  1,
		FeeType: exchange.CryptocurrencyTradeFee,
		Pair: currency.NewPairWithDelimiter(currency.BTC.String(),
			currency.USDT.String(),
			"-"),
		PurchasePrice:       1,
		FiatCurrency:        currency.USD,
		BankTransactionType: exchange.WireTransfer,
	})
	assert.NoError(t, err)
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.ValidateAPICredentials(contextGenerate(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	pair := currency.NewPair(currency.BTC, currency.USDT)
	startTime := time.Date(2021, 2, 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 0, 100)
	_, err := ok.GetHistoricCandles(contextGenerate(), pair, asset.Spot, kline.OneDay, startTime, endTime)
	assert.NoError(t, err, err)

	_, err = ok.GetHistoricCandles(contextGenerate(), pair, asset.Spot, kline.Interval(time.Hour*4), startTime, endTime)
	if !errors.Is(err, kline.ErrRequestExceedsExchangeLimits) {
		t.Errorf("received: '%v' but expected: '%v'", err, kline.ErrRequestExceedsExchangeLimits)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	currencyPair := currency.NewPair(currency.BTC, currency.USDT)
	_, err := ok.GetHistoricCandlesExtended(contextGenerate(), currencyPair, asset.Spot, kline.OneMin, time.Now().Add(-time.Hour), time.Now())
	assert.NoError(t, err)
}

func TestCalculateUpdateOrderbookChecksum(t *testing.T) {
	t.Parallel()

	var orderbookBase orderbook.Base
	err := json.Unmarshal([]byte(calculateOrderbookChecksumUpdateorderbookJSON), &orderbookBase)
	assert.NoError(t, err)
	err = ok.CalculateUpdateOrderbookChecksum(&orderbookBase, 2832680552)
	assert.NoError(t, err)
}

func TestOrderPushData(t *testing.T) {
	t.Parallel()
	n := new(Okx)
	sharedtestvalues.TestFixtureToDataHandler(t, ok, n, "testdata/wsOrders.json", n.WsHandleData)
	seen := 0
	for reading := true; reading; {
		select {
		default:
			reading = false
		case resp := <-n.GetBase().Websocket.DataHandler:
			seen++
			switch v := resp.(type) {
			case *order.Detail:
				switch seen {
				case 1:
					assert.Equal(t, "452197707845865472", v.OrderID, "OrderID")
					assert.Equal(t, "HamsterParty14", v.ClientOrderID, "ClientOrderID")
					assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
					assert.Equal(t, order.Sell, v.Side, "Side")
					assert.Equal(t, order.Filled, v.Status, "Status")
					assert.Equal(t, order.Limit, v.Type, "Type")
					assert.Equal(t, currency.NewPairWithDelimiter("BTC", "USDT", "-"), v.Pair, "Pair")
					assert.Equal(t, 31527.1, v.AverageExecutedPrice, "AverageExecutedPrice")
					assert.Equal(t, time.UnixMilli(1654084334977), v.Date, "Date")
					assert.Equal(t, time.UnixMilli(1654084353263), v.CloseTime, "CloseTime")
					assert.Equal(t, 0.001, v.Amount, "Amount")
					assert.Equal(t, 0.001, v.ExecutedAmount, "ExecutedAmount")
					assert.Equal(t, 0.000, v.RemainingAmount, "RemainingAmount")
					assert.Equal(t, 31527.1, v.Price, "Price")
					assert.Equal(t, 0.02522168, v.Fee, "Fee")
					assert.Equal(t, currency.USDT, v.FeeAsset, "FeeAsset")
				case 2:
					assert.Equal(t, "620258920632008725", v.OrderID, "OrderID")
					assert.Equal(t, asset.Spot, v.AssetType, "AssetType")
					assert.Equal(t, order.Market, v.Type, "Type")
					assert.Equal(t, order.Sell, v.Side, "Side")
					assert.Equal(t, order.Active, v.Status, "Status")
					assert.Equal(t, 0.0, v.Amount, "Amount should be 0 for a market sell")
					assert.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
				case 3:
					assert.Equal(t, "620258920632008725", v.OrderID, "OrderID")
					assert.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
					assert.Equal(t, 0.00038127046945832905, v.Amount, "Amount")
					assert.Equal(t, 0.010000249968, v.Fee, "Fee")
					assert.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount")
					assert.Equal(t, 0.00038128, v.ExecutedAmount, "ExecutedAmount")
					assert.Equal(t, order.PartiallyFilled, v.Status, "Status")
				case 4:
					assert.Equal(t, "620258920632008725", v.OrderID, "OrderID")
					assert.Equal(t, 10.0, v.QuoteAmount, "QuoteAmount")
					assert.Equal(t, 0.010000249968, v.Fee, "Fee")
					assert.Equal(t, 0.0, v.RemainingAmount, "RemainingAmount")
					assert.Equal(t, 0.00038128, v.ExecutedAmount, "ExecutedAmount")
					assert.Equal(t, 0.00038128, v.Amount, "Amount should be derived because order filled")
					assert.Equal(t, order.Filled, v.Status, "Status")
				}
			case error:
				t.Error(v)
			default:
				t.Errorf("Got unexpected data: %T %v", v, v)
			}
		}
	}
	assert.Equal(t, 4, seen, "Saw 4 records")
}

var pushDataMap = map[string]string{
	"Algo Orders":                           `{"arg": {"channel": "orders-algo","uid": "77982378738415879","instType": "FUTURES","instId": "BTC-USD-200329"},"data": [{"instType": "FUTURES","instId": "BTC-USD-200329","ordId": "312269865356374016","ccy": "BTC","algoId": "1234","px": "999","sz": "3","tdMode": "cross","tgtCcy": "","notionalUsd": "","ordType": "trigger","side": "buy","posSide": "long","state": "live","lever": "20","tpTriggerPx": "","tpTriggerPxType": "","tpOrdPx": "","slTriggerPx": "","slTriggerPxType": "","triggerPx": "99","triggerPxType": "last","ordPx": "12","actualSz": "","actualPx": "","tag": "adadadadad","actualSide": "","triggerTime": "1597026383085","cTime": "1597026383000"}]}`,
	"Advanced Algo Order":                   `{"arg": {"channel":"algo-advance","uid": "77982378738415879","instType":"SPOT","instId":"BTC-USDT"},"data":[{"actualPx":"","actualSide":"","actualSz":"0","algoId":"355056228680335360","cTime":"1630924001545","ccy":"","count":"1","instId":"BTC-USDT","instType":"SPOT","lever":"0","notionalUsd":"","ordPx":"","ordType":"iceberg","pTime":"1630924295204","posSide":"net","pxLimit":"10","pxSpread":"1","pxVar":"","side":"buy","slOrdPx":"","slTriggerPx":"","state":"pause","sz":"0.1","szLimit":"0.1","tdMode":"cash","timeInterval":"","tpOrdPx":"","tpTriggerPx":"","tag": "adadadadad","triggerPx":"","triggerTime":"","callbackRatio":"","callbackSpread":"","activePx":"","moveTriggerPx":""}]}`,
	"Position Risk":                         `{"arg": {"channel": "liquidation-warning","uid": "77982378738415879","instType": "FUTURES"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`,
	"Account Greeks":                        `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"","vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`,
	"Rfqs":                                  `{"arg": {"channel": "account-greeks","ccy": "BTC"},"data": [{"thetaBS": "","thetaPA":"","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","vegaBS":"","vegaPA":"","ccy":"BTC","ts":"1620282889345"}]}`,
	"Accounts":                              `{"arg": {"channel": "account","ccy": "BTC","uid": "77982378738415879"},	"data": [{"uTime": "1597026383085","totalEq": "41624.32","isoEq": "3624.32","adjEq": "41624.32","ordFroz": "0","imr": "4162.33","mmr": "4","notionalUsd": "","mgnRatio": "41624.32","details": [{"availBal": "","availEq": "1","ccy": "BTC","cashBal": "1","uTime": "1617279471503","disEq": "50559.01","eq": "1","eqUsd": "45078.3790756226851775","frozenBal": "0","interest": "0","isoEq": "0","liab": "0","maxLoan": "","mgnRatio": "","notionalLever": "0.0022195262185864","ordFrozen": "0","upl": "0","uplLiab": "0","crossLiab": "0","isoLiab": "0","coinUsdPrice": "60000","stgyEq":"0","spotInUseAmt":"","isoUpl":""}]}]}`,
	"Quotes":                                `{"arg": {"channel":"quotes"},"data":[{"validUntil":"1608997227854","uTime":"1608267227834","cTime":"1608267227834","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-220114-25000-C","side":"sell","tgtCcy":""},{"px":"0.0045","sz":"25","instId":"BTC-USD-220114-35000-C","side":"buy","tgtCcy":""}],"quoteId":"25092","rfqId":"18753","traderCode":"SATS","quoteSide":"sell","state":"canceled","clQuoteId":""}]}`,
	"Structure Block Trades":                `{"arg": {"channel":"struc-block-trades"},"data":[{"cTime":"1608267227834","rfqId":"18753","clRfqId":"","quoteId":"25092","clQuoteId":"","blockTdId":"180184","tTraderCode":"ANAND","mTraderCode":"WAGMI","legs":[{"px":"0.0023","sz":"25.0","instId":"BTC-USD-20220630-60000-C","side":"sell","fee":"0.1001","feeCcy":"BTC","tradeId":"10211","tgtCcy":""},{"px":"0.0033","sz":"25","instId":"BTC-USD-20220630-50000-C","side":"buy","fee":"0.1001","feeCcy":"BTC","tradeId":"10212","tgtCcy":""}]}]}`,
	"Spot Grid Algo Orders":                 `{"arg": {"channel": "grid-orders-spot","instType": "ANY"},"data": [{"algoId": "448965992920907776","algoOrdType": "grid","annualizedRate": "0","arbitrageNum": "0","baseSz": "0","cTime": "1653313834104","cancelType": "0","curBaseSz": "0.001776289214","curQuoteSz": "46.801755866","floatProfit": "-0.4953878967772","gridNum": "6","gridProfit": "0","instId": "BTC-USDC","instType": "SPOT","investment": "100","maxPx": "33444.8","minPx": "24323.5","pTime": "1653476023742","perMaxProfitRate": "0.060375293181491054543","perMinProfitRate": "0.0455275366818586","pnlRatio": "0","quoteSz": "100","runPx": "30478.1","runType": "1","singleAmt": "0.00059261","slTriggerPx": "","state": "running","stopResult": "0","stopType": "0","totalAnnualizedRate": "-0.9643551057262827","totalPnl": "-0.4953878967772","tpTriggerPx": "","tradeNum": "3","triggerTime": "1653378736894","uTime": "1653378736894"}]}`,
	"Contract Grid Algo Orders":             `{"arg": {"channel": "grid-orders-contract","instType": "ANY"},"data": [{"actualLever": "1.02","algoId": "449327675342323712","algoOrdType": "contract_grid","annualizedRate": "0.7572437878956523","arbitrageNum": "1","basePos": true,"cTime": "1653400065912","cancelType": "0","direction": "long","eq": "10129.419829834853","floatProfit": "109.537858234853","gridNum": "50","gridProfit": "19.8819716","instId": "BTC-USDT-SWAP","instType": "SWAP","investment": "10000","lever": "5","liqPx": "603.2149534767834","maxPx": "100000","minPx": "10","pTime": "1653484573918","perMaxProfitRate": "995.7080916791230692","perMinProfitRate": "0.0946277854875634","pnlRatio": "0.0129419829834853","runPx": "29216.3","runType": "1","singleAmt": "1","slTriggerPx": "","state": "running","stopType": "0","sz": "10000","tag": "","totalAnnualizedRate": "4.929207431970923","totalPnl": "129.419829834853","tpTriggerPx": "","tradeNum": "37","triggerTime": "1653400066940","uTime": "1653484573589","uly": "BTC-USDT"}]}`,
	"Grid Positions":                        `{"arg": {"channel": "grid-positions","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"adl": "1","algoId": "449327675342323712","avgPx": "29181.4638888888888895","cTime": "1653400065917","ccy": "USDT","imr": "2089.2690000000002","instId": "BTC-USDT-SWAP","instType": "SWAP","last": "29852.7","lever": "5","liqPx": "604.7617536513744","markPx": "29849.7","mgnMode": "cross","mgnRatio": "217.71740878394456","mmr": "41.78538","notionalUsd": "10435.794191550001","pTime": "1653536068723","pos": "35","posSide": "net","uTime": "1653445498682","upl": "232.83263888888962","uplRatio": "0.1139826489932205"}]}`,
	"Grid Sub Orders":                       `{"arg": {"channel": "grid-sub-orders","uid": "44705892343619584","algoId": "449327675342323712"},"data": [{"accFillSz": "0","algoId": "449327675342323712","algoOrdType": "contract_grid","avgPx": "0","cTime": "1653445498664","ctVal": "0.01","fee": "0","feeCcy": "USDT","groupId": "-1","instId": "BTC-USDT-SWAP","instType": "SWAP","lever": "5","ordId": "449518234142904321","ordType": "limit","pTime": "1653486524502","pnl": "","posSide": "net","px": "28007.2","side": "buy","state": "live","sz": "1","tag":"","tdMode": "cross","uTime": "1653445498674"}]}`,
	"Instrument":                            `{"arg": {"channel": "instruments","instType": "FUTURES"},"data": [{"instType": "FUTURES","instId": "BTC-USD-191115","uly": "BTC-USD","category": "1","baseCcy": "","quoteCcy": "","settleCcy": "BTC","ctVal": "10","ctMult": "1","ctValCcy": "USD","optType": "","stk": "","listTime": "","expTime": "","tickSz": "0.01","lotSz": "1","minSz": "1","ctType": "linear","alias": "this_week","state": "live","maxLmtSz":"10000","maxMktSz":"99999","maxTwapSz":"99999","maxIcebergSz":"99999","maxTriggerSz":"9999","maxStopSz":"9999"}]}`,
	"Open Interest":                         `{"arg": {"channel": "open-interest","instId": "LTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","oi": "5000","oiCcy": "555.55","ts": "1597026383085"}]}`,
	"Trade":                                 `{"arg": {"channel": "trades","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","tradeId": "130639474","px": "42219.9","sz": "0.12060306","side": "buy","ts": "1630048897897"}]}`,
	"Estimated Delivery And Exercise Price": `{"arg": {"args": "estimated-price","instType": "FUTURES","uly": "BTC-USD"},"data": [{"instType": "FUTURES","instId": "BTC-USD-170310","settlePx": "200","ts": "1597026383085"}]}`,
	"Mark Price":                            `{"arg": {"channel": "mark-price","instId": "LTC-USD-190628"},"data": [{"instType": "FUTURES","instId": "LTC-USD-190628","markPx": "0.1","ts": "1597026383085"}]}`,
	"Mark Price Candlestick":                `{"arg": {"channel": "mark-price-candle1D","instId": "BTC-USD-190628"},"data": [["1597026383085", "3.721", "3.743", "3.677", "3.708"],["1597026383085", "3.731", "3.799", "3.494", "3.72"]]}`,
	"Price Limit":                           `{"arg": {"channel": "price-limit","instId": "LTC-USD-190628"},"data": [{"instId": "LTC-USD-190628","buyLmt": "200","sellLmt": "300","ts": "1597026383085"}]}`,
	"Test Snapshot Orderbook":               `{"arg": {"channel":"books","instId":"BTC-USDT"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`,
	"Options Trades":                        `{"arg": {"channel": "option-trades", "instType": "OPTION", "instFamily": "BTC-USD" }, "data": [ { "fillVol": "0.5066007836914062", "fwdPx": "16469.69928595038", "idxPx": "16537.2", "instFamily": "BTC-USD", "instId": "BTC-USD-230224-18000-C", "markPx": "0.04690107010619562", "optType": "C", "px": "0.045", "side": "sell", "sz": "2", "tradeId": "38", "ts": "1672286551080" } ] }`,
	"Public Block Trades":                   `{"arg": {"channel":"public-block-trades", "instId":"BTC-USD-231020-5000-P" }, "data":[ { "fillVol":"5", "fwdPx":"26808.16", "idxPx":"27222.5", "instId":"BTC-USD-231020-5000-P", "markPx":"0.0022406326071111", "px":"0.0048", "side":"buy", "sz":"1", "tradeId":"633971452580106242", "ts":"1697422572972"}]}`,
	"Option Summary":                        `{"arg": {"channel": "opt-summary","uly": "BTC-USD"},"data": [{"instType": "OPTION","instId": "BTC-USD-200103-5500-C","uly": "BTC-USD","delta": "0.7494223636","gamma": "-0.6765419039","theta": "-0.0000809873","vega": "0.0000077307","deltaBS": "0.7494223636","gammaBS": "-0.6765419039","thetaBS": "-0.0000809873","vegaBS": "0.0000077307","realVol": "0","bidVol": "","askVol": "1.5625","markVol": "0.9987","lever": "4.0342","fwdPx": "39016.8143629068452065","ts": "1597026383085"}]}`,
	"Funding Rate":                          `{"arg": {"channel": "funding-rate","instId": "BTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "BTC-USD-SWAP","fundingRate": "0.018","nextFundingRate": "","fundingTime": "1597026383085"}]}`,
	"Index Candlestick":                     `{"arg": {"channel": "index-candle30m","instId": "BTC-USDT"},"data": [["1597026383085", "3811.31", "3811.31", "3811.31", "3811.31"]]}`,
	"Index Ticker":                          `{"arg": {"channel": "index-tickers","instId": "BTC-USDT"},"data": [{"instId": "BTC-USDT","idxPx": "0.1","high24h": "0.5","low24h": "0.1","open24h": "0.1","sodUtc0": "0.1","sodUtc8": "0.1","ts": "1597026383085"}]}`,
	"Status":                                `{"arg": {"channel": "status"},"data": [{"title": "Spot System Upgrade","state": "scheduled","begin": "1610019546","href": "","end": "1610019546","serviceType": "1","system": "classic","scheDesc": "","ts": "1597026383085"}]}`,
	"Public Struct Block Trades":            `{"arg": {"channel":"public-struc-block-trades"},"data":[{"cTime":"1608267227834","blockTdId":"1802896","legs":[{"px":"0.323","sz":"25.0","instId":"BTC-USD-20220114-13250-C","side":"sell","tradeId":"15102"},{"px":"0.666","sz":"25","instId":"BTC-USD-20220114-21125-C","side":"buy","tradeId":"15103"}]}]}`,
	"Block Ticker":                          `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`,
	"Account":                               `{"arg": {"channel": "block-tickers"},"data": [{"instType": "SWAP","instId": "LTC-USD-SWAP","volCcy24h": "0","vol24h": "0","ts": "1597026383085"}]}`,
	"Position":                              `{"arg": {"channel":"positions","instType":"FUTURES"},"data":[{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`,
	"Position Data With Underlying":         `{"arg": {"channel": "positions","uid": "77982378738415879","instType": "FUTURES"},"data": [{"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-210430","instType":"FUTURES","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}, {"adl":"1","availPos":"1","avgPx":"2566.31","cTime":"1619507758793","ccy":"ETH","deltaBS":"","deltaPA":"","gammaBS":"","gammaPA":"","imr":"","instId":"ETH-USD-SWAP","instType":"SWAP","interest":"0","last":"2566.22","usdPx":"","lever":"10","liab":"","liabCcy":"","liqPx":"2352.8496681818233","markPx":"2353.849","margin":"0.0003896645377994","mgnMode":"isolated","mgnRatio":"11.731726509588816","mmr":"0.0000311811092368","notionalUsd":"2276.2546609009605","optVal":"","pTime":"1619507761462","pos":"1","posCcy":"","posId":"307173036051017730","posSide":"long","thetaBS":"","thetaPA":"","tradeId":"109844","uTime":"1619507761462","upl":"-0.0000009932766034","uplRatio":"-0.0025490556801078","vegaBS":"","vegaPA":""}]}`,
	"Balance And Position":                  `{"arg": {"channel": "balance_and_position","uid": "77982378738415879"},"data": [{"pTime": "1597026383085","eventType": "snapshot","balData": [{"ccy": "BTC","cashBal": "1","uTime": "1597026383085"}],"posData": [{"posId": "1111111111","tradeId": "2","instId": "BTC-USD-191018","instType": "FUTURES","mgnMode": "cross","posSide": "long","pos": "10","ccy": "BTC","posCcy": "","avgPx": "3320","uTIme": "1597026383085"}]}]}`,
	"Deposit Info Details":                  `{"arg": {"channel": "deposit-info", "uid": "289320****60975104" }, "data": [{ "actualDepBlkConfirm": "0", "amt": "1", "areaCodeFrom": "", "ccy": "USDT", "chain": "USDT-TRC20", "depId": "88165462", "from": "", "fromWdId": "", "pTime": "1674103661147", "state": "0", "subAcct": "test", "to": "TEhFAqpuHa3LY*****8ByNoGnrmexeGMw", "ts": "1674103661123", "txId": "bc5376817*****************dbb0d729f6b", "uid": "289320****60975104" }] }`,
	"Withdrawal Info Details":               `{"arg": {"channel": "deposit-info", "uid": "289320****60975104" }, "data": [{ "actualDepBlkConfirm": "0", "amt": "1", "areaCodeFrom": "", "ccy": "USDT", "chain": "USDT-TRC20", "depId": "88165462", "from": "", "fromWdId": "", "pTime": "1674103661147", "state": "0", "subAcct": "test", "to": "TEhFAqpuHa3LY*****8ByNoGnrmexeGMw", "ts": "1674103661123", "txId": "bc5376817*****************dbb0d729f6b", "uid": "289320****60975104" }] }`,
	"Recurring Buy Order":                   `{"arg": {"channel": "algo-recurring-buy", "instType": "SPOT", "uid": "447*******584" }, "data": [{ "algoClOrdId": "", "algoId": "644497312047435776", "algoOrdType": "recurring", "amt": "100", "cTime": "1699932133373", "cycles": "0", "instType": "SPOT", "investmentAmt": "0", "investmentCcy": "USDC", "mktCap": "0", "nextInvestTime": "1699934415300", "pTime": "1699933314691", "period": "hourly", "pnlRatio": "0", "recurringDay": "", "recurringHour": "1", "recurringList": [{ "avgPx": "0", "ccy": "BTC", "profit": "0", "px": "36482", "ratio": "0.2", "totalAmt": "0" }, { "avgPx": "0", "ccy": "ETH", "profit": "0", "px": "2057.54", "ratio": "0.8", "totalAmt": "0" }], "recurringTime": "12", "state": "running", "stgyName": "stg1", "tag": "", "timeZone": "8", "totalAnnRate": "0", "totalPnl": "0", "uTime": "1699932136249" }] }`,
	"Liquidation Orders":                    `{"arg": {"channel": "liquidation-orders", "instType": "SWAP" }, "data": [ { "details": [ { "bkLoss": "0", "bkPx": "0.007831", "ccy": "", "posSide": "short", "side": "buy", "sz": "13", "ts": "1692266434010" } ], "instFamily": "IOST-USDT", "instId": "IOST-USDT-SWAP", "instType": "SWAP", "uly": "IOST-USDT"}]}`,
	"Economic Calendar":                     `{"arg": {"channel": "economic-calendar" }, "data": [ { "calendarId": "319275", "date": "1597026383085", "region": "United States", "category": "Manufacturing PMI", "event": "S&P Global Manufacturing PMI Final", "refDate": "1597026383085", "actual": "49.2", "previous": "47.3", "forecast": "49.3", "importance": "2", "prevInitial": "", "ccy": "", "unit": "", "ts": "1698648096590" } ] }`,
}

func TestPushData(t *testing.T) {
	t.Parallel()
	for x := range pushDataMap {
		if err := ok.WsHandleData([]byte(pushDataMap[x])); err != nil {
			t.Errorf("Okx %s error %v", x, err)
		}
	}
}

func TestPushDataDynamic(t *testing.T) {
	t.Parallel()
	dataMap := map[string]string{
		"Ticker":             `{"arg": {"channel": "tickers","instId": "BTC-USD-SWAP"},"data": [{"instType": "SWAP","instId": "BTC-USD-SWAP","last": "9999.99","lastSz": "0.1","askPx": "9999.99","askSz": "11","bidPx": "8888.88","bidSz": "5","open24h": "9000","high24h": "10000","low24h": "8888.88","volCcy24h": "2222","vol24h": "2222","sodUtc0": "2222","sodUtc8": "2222","ts": "1597026383085"}]}`,
		"Candlesticks":       `{"arg": {"channel": "candle1D","instId": "BTC-USD-SWAP"},"data": [["1597026383085","8533.02","8553.74","8527.17","8548.26","45247","529.5858061"]]}`,
		"Snapshot OrderBook": `{"arg":{"channel":"books","instId":"BTC-USD-SWAP"},"action":"snapshot","data":[{"asks":[["0.07026","5","0","1"],["0.07027","765","0","3"],["0.07028","110","0","1"],["0.0703","1264","0","1"],["0.07034","280","0","1"],["0.07035","2255","0","1"],["0.07036","28","0","1"],["0.07037","63","0","1"],["0.07039","137","0","2"],["0.0704","48","0","1"],["0.07041","32","0","1"],["0.07043","3985","0","1"],["0.07057","257","0","1"],["0.07058","7870","0","1"],["0.07059","161","0","1"],["0.07061","4539","0","1"],["0.07068","1438","0","3"],["0.07088","3162","0","1"],["0.07104","99","0","1"],["0.07108","5018","0","1"],["0.07115","1540","0","1"],["0.07129","5080","0","1"],["0.07145","1512","0","1"],["0.0715","5016","0","1"],["0.07171","5026","0","1"],["0.07192","5062","0","1"],["0.07197","1517","0","1"],["0.0726","1511","0","1"],["0.07314","10376","0","1"],["0.07354","1","0","1"],["0.07466","10277","0","1"],["0.07626","269","0","1"],["0.07636","269","0","1"],["0.0809","1","0","1"],["0.08899","1","0","1"],["0.09789","1","0","1"],["0.10768","1","0","1"]],"bids":[["0.07014","56","0","2"],["0.07011","608","0","1"],["0.07009","110","0","1"],["0.07006","1264","0","1"],["0.07004","2347","0","3"],["0.07003","279","0","1"],["0.07001","52","0","1"],["0.06997","91","0","1"],["0.06996","4242","0","2"],["0.06995","486","0","1"],["0.06992","161","0","1"],["0.06991","63","0","1"],["0.06988","7518","0","1"],["0.06976","186","0","1"],["0.06975","71","0","1"],["0.06973","1086","0","1"],["0.06961","513","0","2"],["0.06959","4603","0","1"],["0.0695","186","0","1"],["0.06946","3043","0","1"],["0.06939","103","0","1"],["0.0693","5053","0","1"],["0.06909","5039","0","1"],["0.06888","5037","0","1"],["0.06886","1526","0","1"],["0.06867","5008","0","1"],["0.06846","5065","0","1"],["0.06826","1572","0","1"],["0.06801","1565","0","1"],["0.06748","67","0","1"],["0.0674","111","0","1"],["0.0672","10038","0","1"],["0.06652","1","0","1"],["0.06625","1526","0","1"],["0.06619","10924","0","1"],["0.05986","1","0","1"],["0.05387","1","0","1"],["0.04848","1","0","1"],["0.04363","1","0","1"]],"ts":"1659792392540","checksum":-1462286744}]}`,
	}
	for x := range dataMap {
		if err := ok.WsHandleData([]byte(dataMap[x])); err != nil {
			t.Errorf("Okx %s error %v", x, err)
		}
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetHistoricTrades(contextGenerate(), currency.NewPair(currency.BTC, currency.USDT), asset.Spot, time.Now().Add(-time.Minute*4), time.Now().Add(-time.Minute*2))
	assert.NoError(t, err)
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
	err := ok.InstrumentsSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}

func TestTickersSubscription(t *testing.T) {
	t.Parallel()
	err := ok.TickersSubscription("subscribe", asset.Margin, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	err = ok.TickersSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}
func TestOpenInterestSubscription(t *testing.T) {
	t.Parallel()
	err := ok.OpenInterestSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.BTC, currency.NewCode("USD-SWAP")))
	assert.NoError(t, err)
}
func TestCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.PerpetualSwap)
	assert.NoError(t, err, "couldn't find enabled tradable pairs")
	if len(enabled) == 0 {
		t.SkipNow()
	}
	err = ok.CandlesticksSubscription("subscribe", okxChannelCandle1m, asset.Futures, enabled[0])
	assert.NoError(t, err)
}

func TestTradesSubscription(t *testing.T) {
	t.Parallel()
	err := ok.TradesSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}

func TestEstimatedDeliveryExercisePriceSubscription(t *testing.T) {
	t.Parallel()
	futuresPairs, err := ok.FetchTradablePairs(contextGenerate(), asset.Futures)
	assert.NoErrorf(t, err, "%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	err = ok.EstimatedDeliveryExercisePriceSubscription("subscribe", asset.Futures, futuresPairs[0])
	assert.NoError(t, err)
}

func TestMarkPriceSubscription(t *testing.T) {
	t.Parallel()
	futuresPairs, err := ok.FetchTradablePairs(contextGenerate(), asset.Futures)
	assert.NoErrorf(t, err, "%s error while fetching tradable pairs for instrument type %v: %v", ok.Name, asset.Futures, err)
	if len(futuresPairs) == 0 {
		t.SkipNow()
	}
	err = ok.MarkPriceSubscription("subscribe", asset.Futures, futuresPairs[0])
	assert.NoError(t, err)
}

func TestMarkPriceCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.Spot)
	assert.NoError(t, err, "couldn't find enabled tradable pairs")
	if len(enabled) == 0 {
		t.SkipNow()
	}
	err = ok.MarkPriceCandlesticksSubscription("subscribe", okxChannelMarkPriceCandle1Y, asset.Futures, enabled[0])
	assert.NoError(t, err)
}

func TestPriceLimitSubscription(t *testing.T) {
	t.Parallel()
	err := ok.PriceLimitSubscription("subscribe", asset.PerpetualSwap, currency.NewPairWithDelimiter("BTC", "USD-SWAP", currency.DashDelimiter))
	assert.NoError(t, err)
}

func TestOrderBooksSubscription(t *testing.T) {
	t.Parallel()
	enabled, err := ok.GetEnabledPairs(asset.Spot)
	assert.NoError(t, err, "couldn't find enabled tradable pairs")
	if len(enabled) == 0 {
		t.SkipNow()
	}
	err = ok.OrderBooksSubscription("subscribe", okxChannelOrderBooks, asset.Futures, enabled[0])
	assert.NoError(t, err)
	err = ok.OrderBooksSubscription("unsubscribe", okxChannelOrderBooks, asset.Futures, enabled[0])
	assert.NoError(t, err)
}

func TestOptionSummarySubscription(t *testing.T) {
	t.Parallel()
	err := ok.OptionSummarySubscription("subscribe", currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
	err = ok.OptionSummarySubscription("unsubscribe", currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}

func TestFundingRateSubscription(t *testing.T) {
	t.Parallel()
	err := ok.FundingRateSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")))
	assert.NoError(t, err)
	err = ok.FundingRateSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.BTC, currency.NewCode("USDT-SWAP")))
	assert.NoError(t, err)
}

func TestIndexCandlesticksSubscription(t *testing.T) {
	t.Parallel()
	err := ok.IndexCandlesticksSubscription("subscribe", okxChannelIndexCandle6M, asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
	err = ok.IndexCandlesticksSubscription("unsubscribe", okxChannelIndexCandle6M, asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}
func TestIndexTickerChannelIndexTickerChannel(t *testing.T) {
	t.Parallel()
	err := ok.IndexTickerChannel("subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
	err = ok.IndexTickerChannel("unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}

func TestStatusSubscription(t *testing.T) {
	t.Parallel()
	err := ok.StatusSubscription("subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
	err = ok.StatusSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}

func TestPublicStructureBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	err := ok.PublicStructureBlockTradesSubscription("subscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
	err = ok.PublicStructureBlockTradesSubscription("unsubscribe", asset.Spot, currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}
func TestBlockTickerSubscription(t *testing.T) {
	t.Parallel()
	err := ok.BlockTickerSubscription("subscribe", asset.Options, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	err = ok.BlockTickerSubscription("unsubscribe", asset.Options, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}

func TestPublicBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	err := ok.PublicBlockTradesSubscription("subscribe", asset.Options, currency.NewPairWithDelimiter("BTC", "USDT-SWAP", "-"))
	assert.NoError(t, err)
	err = ok.PublicBlockTradesSubscription("unsubscribe", asset.Options, currency.NewPairWithDelimiter("BTC", "USDT-SWAP", "-"))
	assert.NoError(t, err)
}

// ************ Authenticated Websocket endpoints Test **********************************************

func TestWsAccountSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.WsAccountSubscription("subscribe", asset.Spot, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
}

func TestWsPlaceOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.WsPlaceOrder(&PlaceOrderRequestParam{
		InstrumentID: "BTC-USDC",
		TradeMode:    "cross",
		Side:         "Buy",
		OrderType:    "limit",
		Amount:       2.6,
		Price:        2.1,
		Currency:     "BTC",
	})
	assert.Falsef(t, err != nil && !errors.Is(err, errWebsocketStreamNotAuthenticated), "%s WsPlaceOrder() error: %v", ok.Name, err)
}

func TestWsPlaceMultipleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	var resp []PlaceOrderRequestParam
	err := json.Unmarshal([]byte(placeOrderArgs), &resp)
	assert.NoError(t, err)
	pairs, err := ok.FetchTradablePairs(contextGenerate(), asset.Spot)
	require.NoError(t, err)
	if len(pairs) == 0 {
		t.Skip("no pairs found")
	}
	_, err = ok.WsPlaceMultipleOrder(resp)
	assert.False(t, (err != nil && !errors.Is(err, errWebsocketStreamNotAuthenticated)), err)
}

func TestWsCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.WsCancelOrder(CancelOrderRequestParam{
		InstrumentID: "BTC-USD-190927",
		OrderID:      "2510789768709120",
	})
	assert.NoError(t, err)
}

func TestWsCancleMultipleOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.WsCancelMultipleOrder([]CancelOrderRequestParam{{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
	}})
	assert.Falsef(t, (err != nil && !strings.Contains(err.Error(), "Cancellation failed as the order does not exist.")),
		"Okx WsCancleMultipleOrder() error", err)
}

func TestWsAmendOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.WsAmendOrder(&AmendOrderRequestParams{
		InstrumentID: "DCR-BTC",
		OrderID:      "2510789768709120",
		NewPrice:     1233324.332,
		NewQuantity:  1234,
	})
	assert.True(t, (err == nil || strings.Contains(err.Error(), "order does not exist.")), "%s WsAmendOrder() error %v", ok.Name, err)
}

func TestWsAmendMultipleOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)

	_, err := ok.WsAmendMultipleOrders([]AmendOrderRequestParams{
		{
			InstrumentID: "DCR-BTC",
			OrderID:      "2510789768709120",
			NewPrice:     1233324.332,
			NewQuantity:  1234,
		},
	})
	assert.Truef(t, (err == nil || strings.Contains(err.Error(), "Order modification failed as the order does not exist.")), "%s WsAmendMultipleOrders() %v", ok.Name, err)
}

func TestWsMassCancelOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.WsMassCancelOrders([]CancelMassReqParam{
		{
			InstrumentType:   "OPTION",
			InstrumentFamily: "BTC-USD",
		},
	})
	assert.Truef(t, (err == nil || strings.Contains(err.Error(), "order does not exist.")), "%s WsMassCancelOrders() error %v", ok.Name, err)
}

func TestWsPositionChannel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.WsPositionChannel("subscribe", asset.Options, currency.NewPair(currency.USD, currency.BTC))
	assert.NoError(t, err)
}

func TestBalanceAndPositionSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.BalanceAndPositionSubscription("subscribe", "1234")
	assert.NoError(t, err)
	err = ok.BalanceAndPositionSubscription("unsubscribe", "1234")
	assert.NoError(t, err)
}

func TestWsOrderChannel(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.WsOrderChannel("subscribe", asset.Margin, currency.NewPair(currency.SOL, currency.USDT), "")
	assert.NoError(t, err)
	err = ok.WsOrderChannel("unsubscribe", asset.Margin, currency.NewPair(currency.SOL, currency.USDT), "")
	assert.NoError(t, err)
}

func TestAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.AlgoOrdersSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")))
	assert.NoError(t, err)
	err = ok.AlgoOrdersSubscription("unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")))
	assert.NoError(t, err)
}

func TestAdvanceAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.AdvanceAlgoOrdersSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")), "")
	assert.NoError(t, err)
	err = ok.AdvanceAlgoOrdersSubscription("unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")), "")
	assert.NoError(t, err)
}

func TestPositionRiskWarningSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.PositionRiskWarningSubscription("subscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")))
	assert.NoError(t, err)
	err = ok.PositionRiskWarningSubscription("unsubscribe", asset.PerpetualSwap, currency.NewPair(currency.SOL, currency.NewCode("USD-SWAP")))
	assert.NoError(t, err)
}

func TestAccountGreeksSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.AccountGreeksSubscription("subscribe", currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
	err = ok.AccountGreeksSubscription("unsubscribe", currency.NewPair(currency.SOL, currency.USD))
	assert.NoError(t, err)
}

func TestRfqSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.RfqSubscription("subscribe", "")
	assert.NoError(t, err)
	err = ok.RfqSubscription("unsubscribe", "")
	assert.NoError(t, err)
}

func TestQuotesSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.QuotesSubscription("subscribe")
	assert.NoError(t, err)
	err = ok.QuotesSubscription("unsubscribe")
	assert.NoError(t, err)
}

func TestStructureBlockTradesSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.StructureBlockTradesSubscription("subscribe")
	assert.NoError(t, err)
	err = ok.StructureBlockTradesSubscription("unsubscribe")
	assert.NoError(t, err)
}

func TestSpotGridAlgoOrdersSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.SpotGridAlgoOrdersSubscription("subscribe", asset.Empty, currency.EMPTYPAIR, "")
	assert.NoError(t, err)
	err = ok.SpotGridAlgoOrdersSubscription("unsubscribe", asset.Empty, currency.EMPTYPAIR, "")
	assert.NoError(t, err)
}

func TestContractGridAlgoOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.ContractGridAlgoOrders("subscribe", asset.Empty, currency.EMPTYPAIR, "")
	assert.NoError(t, err)
	err = ok.ContractGridAlgoOrders("unsubscribe", asset.Empty, currency.EMPTYPAIR, "")
	assert.NoError(t, err)
}

func TestGridPositionsSubscription(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	err := ok.GridPositionsSubscription("subscribe", "1234")
	assert.False(t, err != nil && !strings.Contains(err.Error(), "channel:grid-positions doesn't exist"), "%s GridPositionsSubscription() error: %v", ok.Name, err)
	err = ok.GridPositionsSubscription("unsubscribe", "1234")
	assert.False(t, err != nil && !strings.Contains(err.Error(), "channel:grid-positions doesn't exist"), "%s GridPositionsSubscription() error: %v", ok.Name, err)
}

func TestGridSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)

	err := ok.GridSubOrders("subscribe", "")
	assert.False(t, err != nil && !strings.Contains(err.Error(), "grid-sub-orders doesn't exist"), "%s GridSubOrders() error: %v", ok.Name, err)
	err = ok.GridSubOrders("unsubscribe", "")
	assert.False(t, err != nil && !strings.Contains(err.Error(), "grid-sub-orders doesn't exist"), "%s GridSubOrders() error: %v", ok.Name, err)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := ok.GetServerTime(contextGenerate(), asset.Empty)
	assert.NoError(t, err)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetAvailableTransferChains(contextGenerate(), currency.BTC)
	assert.NoError(t, err)
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

			r := ok.GetIntervalEnum(tt.Interval, tt.AppendUTC)
			assert.False(t, r != tt.Expected, "%s: received: %s but expected: %s", tt.Description, r, tt.Expected)
		})
	}
}

func TestInstrument(t *testing.T) {
	t.Parallel()

	var i Instrument
	err := json.Unmarshal([]byte(instrumentJSON), &i)
	assert.NoError(t, err)

	assert.False(t, i.Alias != "", "expected empty alias")
	assert.False(t, i.BaseCurrency != "", "expected empty base currency")
	assert.False(t, i.Category != "1", "expected 1 category")
	assert.False(t, i.ContractMultiplier != 1, "expected 1 contract multiplier")
	assert.False(t, i.ContractType != "linear", "expected linear contract type")
	assert.False(t, i.ContractValue.Float64() != 0.0001, "expected 0.0001 contract value")
	assert.False(t, i.ContractValueCurrency != currency.BTC.String(), "expected BTC contract value currency")
	assert.False(t, !i.ExpTime.Time().IsZero(), "expected empty expiry time")
	assert.False(t, i.InstrumentFamily != "BTC-USDC", "expected BTC-USDC instrument family")
	assert.False(t, i.InstrumentID != "BTC-USDC-SWAP", "expected BTC-USDC-SWAP instrument ID")
	swap := ok.GetInstrumentTypeFromAssetItem(asset.PerpetualSwap)
	assert.False(t, i.InstrumentType != swap, "expected SWAP instrument type")
	assert.False(t, i.MaxLeverage != 125, "expected 125 leverage")
	assert.False(t, i.ListTime.Time().UnixMilli() != 1666076190000, "expected 1666076190000 listing time")
	assert.False(t, i.LotSize != 1, "expected 1 lot size")
	assert.False(t, i.MaxSpotIcebergSize != 100000000.0000000000000000, "expected 100000000.0000000000000000 max iceberg order size")
	assert.False(t, i.MaxQuantityOfSpotLimitOrder != 100000000, "expected 100000000 max limit order size")
	assert.False(t, i.MaxQuantityOfMarketLimitOrder != 85000, "expected 85000 max market order size")
	assert.False(t, i.MaxStopSize != 85000, "expected 85000 max stop order size")
	assert.False(t, i.MaxTriggerSize != 100000000.0000000000000000,
		"expected 100000000.0000000000000000 max trigger order size")
	assert.False(t, i.MaxQuantityOfSpotTwapLimitOrder != 0, "expected empty max TWAP size")
	assert.False(t, i.MinimumOrderSize != 1, "expected 1 min size")
	assert.False(t, i.OptionType != "", "expected empty option type")
	assert.False(t, i.QuoteCurrency != "", "expected empty quote currency")
	assert.False(t, i.SettlementCurrency != currency.USDC.String(), "expected USDC settlement currency")
	assert.False(t, i.State != "live", "expected live state")
	assert.False(t, i.StrikePrice != "", "expected empty strike price")
	assert.False(t, i.TickSize != 0.1, "expected 0.1 tick size")
	assert.False(t, i.Underlying != "BTC-USDC", "expected BTC-USDC underlying")
}

func TestGetLatestFundingRate(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD-SWAP")
	assert.NoError(t, err)
	_, err = ok.GetLatestFundingRates(contextGenerate(), &fundingrate.LatestRateRequest{
		Asset:                asset.PerpetualSwap,
		Pair:                 cp,
		IncludePredictedRate: true,
	})
	assert.NoError(t, err)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	cp, err := currency.NewPairFromString("BTC-USD-SWAP")
	assert.NoError(t, err)
	r := &fundingrate.HistoricalRatesRequest{
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
	_, err = ok.GetHistoricalFundingRates(contextGenerate(), r)
	assert.NoError(t, err)

	r.StartDate = time.Now().Add(-time.Hour * 24 * 120)
	_, err = ok.GetHistoricalFundingRates(contextGenerate(), r)
	assert.True(t, errors.Is(err, fundingrate.ErrFundingRateOutsideLimits), err)

	r.RespectHistoryLimits = true
	_, err = ok.GetHistoricalFundingRates(contextGenerate(), r)
	assert.NoError(t, err)
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	is, err := ok.IsPerpetualFutureCurrency(asset.Binary, currency.NewPair(currency.BTC, currency.USDT))
	assert.NoError(t, err)
	assert.False(t, is, "expected false")

	cp, err := currency.NewPairFromString("BTC-USD-SWAP")
	assert.NoError(t, err)
	is, err = ok.IsPerpetualFutureCurrency(asset.PerpetualSwap, cp)
	assert.NoError(t, err)
	assert.True(t, is, "expected true")
}

func TestGetAssetsFromInstrumentTypeOrID(t *testing.T) {
	t.Parallel()
	_, err := ok.GetAssetsFromInstrumentTypeOrID("", "")
	assert.True(t, errors.Is(err, errEmptyArgument), err)

	assets, err := ok.GetAssetsFromInstrumentTypeOrID("SPOT", "")
	assert.NoError(t, err)
	assert.Falsef(t, len(assets) != 1, "received %v expected %v", len(assets), 1)
	assert.Falsef(t, assets[0] != asset.Spot, "received %v expected %v", assets[0], asset.Spot)

	assets, err = ok.GetAssetsFromInstrumentTypeOrID("", ok.CurrencyPairs.Pairs[asset.Futures].Enabled[0].String())
	assert.NoError(t, err)
	assert.False(t, len(assets) != 1, "received %v expected %v", len(assets), 1)
	assert.Falsef(t, assets[0] != asset.Futures, "received %v expected %v", assets[0], asset.Futures)

	assets, err = ok.GetAssetsFromInstrumentTypeOrID("", ok.CurrencyPairs.Pairs[asset.PerpetualSwap].Enabled[0].String())
	assert.NoError(t, err)
	assert.Falsef(t, len(assets) != 1, "received %v expected %v", len(assets), 1)
	assert.Falsef(t, assets[0] != asset.PerpetualSwap, "received %v expected %v", assets[0], asset.PerpetualSwap)

	_, err = ok.GetAssetsFromInstrumentTypeOrID("", "test")
	assert.True(t, errors.Is(err, currency.ErrCurrencyNotSupported), err)

	_, err = ok.GetAssetsFromInstrumentTypeOrID("", "test-test")
	assert.True(t, errors.Is(err, asset.ErrNotSupported), err)

	assets, err = ok.GetAssetsFromInstrumentTypeOrID("", ok.CurrencyPairs.Pairs[asset.Margin].Enabled[0].String())
	assert.True(t, errors.Is(err, nil), err)
	var found bool
	for i := range assets {
		if assets[i] == asset.Margin {
			found = true
		}
	}
	assert.Truef(t, found, "received %v expected %v", assets, asset.Margin)

	assets, err = ok.GetAssetsFromInstrumentTypeOrID("", ok.CurrencyPairs.Pairs[asset.Spot].Enabled[0].String())
	assert.True(t, errors.Is(err, nil), err)
	found = false
	for i := range assets {
		if assets[i] == asset.Spot {
			found = true
		}
	}
	assert.True(t, found, "received %v expected %v", assets, asset.Spot)
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	err := ok.SetMarginType(contextGenerate(), asset.Spot, currency.NewBTCUSDT(), margin.Isolated)
	assert.Truef(t, errors.Is(err, common.ErrFunctionNotSupported), "received '%v', expected '%v'", err, asset.ErrNotSupported)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	cp, err := currency.NewPairFromString("eth/btc")
	assert.NoError(t, err)
	_, err = ok.ChangePositionMargin(contextGenerate(), &margin.PositionChangeRequest{
		Pair:                    cp,
		Asset:                   asset.Margin,
		MarginType:              margin.Isolated,
		OriginalAllocatedMargin: 4.0695,
		NewAllocatedMargin:      5,
	})
	assert.NoError(t, err)
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetCollateralMode(contextGenerate(), asset.Spot)
	assert.Falsef(t, errors.Is(err, nil), "received '%v', expected '%v'", err, nil)
	_, err = ok.GetCollateralMode(contextGenerate(), asset.Futures)
	assert.True(t, !errors.Is(err, nil), "received '%v', expected '%v'", err, nil)
	_, err = ok.GetCollateralMode(contextGenerate(), asset.USDTMarginedFutures)
	assert.False(t, errors.Is(err, asset.ErrNotSupported), "received '%v', expected '%v'", err, asset.ErrNotSupported)
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	err := ok.SetCollateralMode(contextGenerate(), asset.Spot, collateral.SingleMode)
	assert.Falsef(t, errors.Is(err, common.ErrFunctionNotSupported), "received '%v', expected '%v'", err, asset.ErrNotSupported)
}

func TestGetPositionSummary(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	pp, err := ok.CurrencyPairs.GetPairs(asset.PerpetualSwap, true)
	assert.NoError(t, err)
	_, err = ok.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.PerpetualSwap,
		Pair:           pp[0],
		UnderlyingPair: currency.EMPTYPAIR,
	})
	assert.NoError(t, err)

	pp, err = ok.CurrencyPairs.GetPairs(asset.Futures, true)
	assert.NoError(t, err)
	_, err = ok.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.Futures,
		Pair:           pp[0],
		UnderlyingPair: currency.EMPTYPAIR,
	})
	assert.NoError(t, err)

	_, err = ok.GetFuturesPositionSummary(contextGenerate(), &futures.PositionSummaryRequest{
		Asset:          asset.Spot,
		Pair:           pp[0],
		UnderlyingPair: currency.NewBTCUSDT(),
	})
	assert.True(t, !errors.Is(err, futures.ErrNotFuturesAsset), "received '%v', expected '%v'", err, futures.ErrNotFuturesAsset)
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	pp, err := ok.CurrencyPairs.GetPairs(asset.Futures, true)
	assert.NoError(t, err)
	_, err = ok.GetFuturesPositionOrders(contextGenerate(), &futures.PositionsRequest{
		Asset:     asset.Futures,
		Pairs:     []currency.Pair{pp[0]},
		StartDate: time.Now().Add(time.Hour * 24 * -7),
		EndDate:   time.Now(),
	})
	assert.NoError(t, err)

	_, err = ok.GetFuturesPositionOrders(contextGenerate(), &futures.PositionsRequest{
		Asset:     asset.Spot,
		Pairs:     []currency.Pair{pp[0]},
		StartDate: time.Now().Add(time.Hour * 24 * -7),
	})
	assert.True(t, !errors.Is(err, asset.ErrNotSupported), "received '%v', expected '%v'", err, asset.ErrNotSupported)
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	pp, err := ok.CurrencyPairs.GetPairs(asset.Futures, true)
	assert.NoError(t, err)
	_, err = ok.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Multi, order.UnknownSide)
	assert.NoError(t, err)

	_, err = ok.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.UnknownSide)
	assert.True(t, !errors.Is(err, errOrderSideRequired), "received '%v', expected '%v'", err, errOrderSideRequired)

	_, err = ok.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.Long)
	assert.NoError(t, err)

	_, err = ok.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.Short)
	assert.NoError(t, err)

	_, err = ok.GetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, order.CouldNotBuy)
	assert.True(t, !errors.Is(err, errInvalidOrderSide), "received '%v', expected '%v'", err, errInvalidOrderSide)
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	pp, err := ok.CurrencyPairs.GetPairs(asset.Futures, true)
	assert.NoError(t, err)
	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Multi, 5, order.UnknownSide)
	assert.NoError(t, err)

	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.UnknownSide)
	assert.True(t, !errors.Is(err, errOrderSideRequired), "received '%v', expected '%v'", err, errOrderSideRequired)

	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.Long)
	assert.True(t, err != nil, err)

	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.Short)
	assert.True(t, err != nil, err)

	err = ok.SetLeverage(contextGenerate(), asset.Futures, pp[0], margin.Isolated, 5, order.CouldNotBuy)
	assert.True(t, !errors.Is(err, errInvalidOrderSide), "received '%v', expected '%v'", err, errInvalidOrderSide)

	err = ok.SetLeverage(contextGenerate(), asset.Spot, pp[0], margin.Multi, 5, order.UnknownSide)
	assert.True(t, !errors.Is(err, asset.ErrNotSupported), "received '%v', expected '%v'", err, asset.ErrNotSupported)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := ok.GetFuturesContractDetails(context.Background(), asset.Spot)
	assert.False(t, !errors.Is(err, futures.ErrNotFuturesAsset), err)
	_, err = ok.GetFuturesContractDetails(context.Background(), asset.USDTMarginedFutures)
	assert.False(t, !errors.Is(err, asset.ErrNotSupported), err)

	_, err = ok.GetFuturesContractDetails(context.Background(), asset.Futures)
	assert.False(t, !errors.Is(err, nil), err)
	_, err = ok.GetFuturesContractDetails(context.Background(), asset.PerpetualSwap)
	assert.False(t, !errors.Is(err, nil), err)
}

func TestWsProcessOrderbook5(t *testing.T) {
	t.Parallel()

	var ob5payload = []byte(`{"arg":{"channel":"books5","instId":"OKB-USDT"},"data":[{"asks":[["0.0000007465","2290075956","0","4"],["0.0000007466","1747284705","0","4"],["0.0000007467","1338861655","0","3"],["0.0000007468","1661668387","0","6"],["0.0000007469","2715477116","0","5"]],"bids":[["0.0000007464","15693119","0","1"],["0.0000007463","2330835024","0","4"],["0.0000007462","1182926517","0","2"],["0.0000007461","3818684357","0","4"],["0.000000746","6021641435","0","7"]],"instId":"OKB-USDT","ts":"1695864901807","seqId":4826378794}]}`)
	err := ok.wsProcessOrderbook5(ob5payload)
	assert.NoError(t, err)
	required := currency.NewPairWithDelimiter("OKB", "USDT", "-")

	got, err := orderbook.Get("okx", required, asset.Spot)
	require.NoError(t, err)

	assert.False(t, len(got.Asks) != 5, "expected %v, received %v", 5, len(got.Asks))
	assert.False(t, len(got.Bids) != 5, "expected %v, received %v", 5, len(got.Bids))
	// Book replicated to margin
	got, err = orderbook.Get("okx", required, asset.Margin)
	require.NoError(t, err)
	assert.Falsef(t, len(got.Asks) != 5, "expected %v, received %v", 5, len(got.Asks))
	assert.Falsef(t, len(got.Bids) != 5, "expected %v, received %v", 5, len(got.Bids))
}

func TestGetLeverateEstimatedInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetLeverateEstimatedInfo(context.Background(), "MARGIN", "cross", "1", "", "BTC-USDT", currency.BTC)
	assert.NoError(t, err)
}

func TestManualBorrowAndRepayInQuickMarginMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.ManualBorrowAndRepayInQuickMarginMode(context.Background(), &BorrowAndRepay{
		Amount:       1,
		InstrumentID: "BTC-USDT",
		LoanCcy:      "USDT",
		Side:         "borrow",
	})
	assert.NoError(t, err)
}

func TestGetBorrowAndRepayHistoryInQuickMarginMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetBorrowAndRepayHistoryInQuickMarginMode(context.Background(), currency.EMPTYPAIR, currency.BTC, "borrow", "", "", time.Time{}, time.Time{}, 10)
	assert.NoError(t, err)
}

func TestGetVIPInterestAccruedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetVIPInterestAccruedData(context.Background(), currency.ETH, "", time.Time{}, time.Time{}, 10)
	assert.NoError(t, err)
}
func TestGetVIPInterestDeductedData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetVIPInterestDeductedData(context.Background(), currency.ETH, "", time.Time{}, time.Time{}, 10)
	assert.NoError(t, err)
}

func TestGetVIPLoanOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetVIPLoanOrderList(context.Background(), "", "1", currency.BTC, time.Time{}, time.Now(), 20)
	assert.NoError(t, err)
}

func TestGetVIPLoanOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetVIPLoanOrderDetail(context.Background(), "123456", currency.BTC, time.Time{}, time.Time{}, 10)
	assert.NoError(t, err)
}
func TestSetRiskOffsetType(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.SetRiskOffsetType(context.Background(), "3")
	assert.NoError(t, err)
}

func TestActivateOption(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.ActivateOption(context.Background())
	assert.NoError(t, err)
}

func TestSetAutoLoan(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.SetAutoLoan(context.Background(), true)
	assert.NoError(t, err)
}

func TestSetAccountMode(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.SetAccountMode(context.Background(), "1")
	assert.NoError(t, err)
}

func TestResetMMPStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.ResetMMPStatus(contextGenerate(), "OPTION", "BTC-USD")
	assert.True(t, err != nil && !strings.Contains(err.Error(), "No permission to use this API"), "%s ResetMMPStatus() error %v", ok.Name, err)
}

func TestSetMMP(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.SetMMP(context.Background(), &MMPConfig{
		InstrumentFamily: "BTC-USD",
		TimeInterval:     5000,
		FrozenInterval:   2000,
		QuantityLimit:    100,
	})
	assert.NoError(t, err)
}

func TestGetMMPConfig(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetMMPConfig(context.Background(), "BTC-USD")
	assert.NoError(t, err)
}

func TestMassCancelOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.MassCancelOrder(context.Background(), "OPTION", "BTC-USD")
	assert.NoError(t, err)
}

func TestCancelAllMMPOrdersAfterCountdown(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.CancelAllMMPOrdersAfterCountdown(context.Background(), "60")
	assert.NoError(t, err)
}

func TestAmendAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.AmendAlgoOrder(context.Background(), &AmendAlgoOrderParam{
		AlgoID:       "2510789768709120",
		InstrumentID: "BTC-USDT-SWAP",
		NewSize:      2,
	})
	assert.NoError(t, err)
}

func TestGetAlgoOrderDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetAlgoOrderDetail(context.Background(), "1234231231423", "")
	assert.NoError(t, err)
}

func TestClosePositionForContractrid(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.ClosePositionForContractrid(context.Background(), &ClosePositionParams{
		AlgoID:                  "448965992920907776",
		MarketCloseAllPositions: true,
	})
	assert.NoError(t, err)
}

func TestCancelClosePositionOrderForContractGrid(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.CancelClosePositionOrderForContractGrid(context.Background(), &CancelClosePositionOrder{
		AlgoID:  "448965992920907776",
		OrderID: "570627699870375936",
	})
	assert.NoError(t, err)
}

func TestInstantTriggerGridAlgoOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.InstantTriggerGridAlgoOrder(context.Background(), "123456789")
	assert.NoError(t, err)
}

func TestComputeMinInvestment(t *testing.T) {
	t.Parallel()
	_, err := ok.ComputeMinInvestment(context.Background(), &ComputeInvestmentDataParam{
		InstrumentID:  "ETH-USDT",
		AlgoOrderType: "grid",
		GridNumber:    50,
		MaxPrice:      5000,
		MinPrice:      3000,
		RunType:       "1",
		InvestmentData: []InvestmentData{
			{
				Amount:   0.01,
				Currency: "ETH",
			},
			{
				Amount:   100,
				Currency: "USDT",
			},
		},
	})
	assert.NoError(t, err)
}

func TestRSIBackTesting(t *testing.T) {
	t.Parallel()
	_, err := ok.RSIBackTesting(context.Background(), "BTC-USDT", "", "", 50, 14, kline.FiveMin)
	assert.NoError(t, err)
}

func TestSignalBotTrading(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.GetSignalBotOrderDetail(context.Background(), "contract", "623833708424069120")
	assert.NoError(t, err)
}

func TestGetSignalOrderPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetSignalOrderPositions(context.Background(), "contract", "623833708424069120")
	assert.NoError(t, err)
}

func TestGetSignalBotSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetSignalBotSubOrders(context.Background(), "623833708424069120", "contract", "filled", "", "", "", time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGetSignalBotEventHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetSignalBotEventHistory(context.Background(), "12345", time.Time{}, time.Now(), 50)
	assert.NoError(t, err)
}

func TestPlaceRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.PlaceRecurringBuyOrder(context.Background(), &PlaceRecurringBuyOrderParam{
		StrategyName: "BTC|ETH recurring buy monthly",
		Amount:       100,
		RecurringList: []RecurringListItem{
			{
				Currency: currency.BTC,
				Ratio:    0.2,
			},
			{
				Currency: currency.ETH,
				Ratio:    0.8,
			},
		},
		Period:             "monthly",
		RecurringDay:       "1",
		RecurringTime:      0,
		TimeZone:           "8", // UTC +8
		TradeMode:          "cross",
		InvestmentCurrency: "USDT",
	})
	assert.NoError(t, err)
}

func TestAmendRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.AmendRecurringBuyOrder(context.Background(), &AmendRecurringOrderParam{
		AlgoID:       "448965992920907776",
		StrategyName: "stg1",
	})
	assert.NoError(t, err)
}

func TestStopRecurringBuyOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.StopRecurringBuyOrder(context.Background(), []StopRecurringBuyOrder{{AlgoID: "1232323434234"}})
	assert.NoError(t, err)
}

func TestGetRecurringBuyOrderList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetRecurringBuyOrderList(context.Background(), "", time.Time{}, time.Time{}, 30)
	assert.NoError(t, err)
}

func TestGetRecurringBuyOrderHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetRecurringBuyOrderHistory(context.Background(), "", time.Time{}, time.Time{}, 30)
	assert.NoError(t, err)
}

func TestGetRecurringOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetRecurringOrderDetails(context.Background(), "560473220642766848")
	assert.NoError(t, err)
}

func TestGetRecurringSubOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetRecurringSubOrders(context.Background(), "560473220642766848", "123422", time.Time{}, time.Now(), 0)
	assert.NoError(t, err)
}

func TestGetExistingLeadingPositions(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetExistingLeadingPositions(context.Background(), "SPOT", "BTC-USDT", time.Now(), time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGetLeadingPositionsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetLeadingPositionsHistory(context.Background(), "OPTION", "", time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
}

func TestPlaceLeadingStopOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.PlaceLeadingStopOrder(context.Background(), &TPSLOrderParam{
		SubPositionID:          "1235454",
		TakeProfitTriggerPrice: 123455})
	assert.NoError(t, err)
}

func TestCloseLeadingPosition(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.CloseLeadingPosition(context.Background(), &CloseLeadingPositionParam{})
	assert.Truef(t, !errors.Is(err, errNilArgument), "expected %v, got %v", errNilArgument, err)
	_, err = ok.CloseLeadingPosition(context.Background(), &CloseLeadingPositionParam{
		SubPositionID: "518541406042591232",
	})
	assert.NoError(t, err)
}

func TestGetLeadingInstrument(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetLeadingInstrument(context.Background(), "SWAP")
	assert.NoError(t, err)
}

func TestAmendLeadingInstruments(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.AmendLeadingInstruments(context.Background(), "BTC-USDT-SWAP", "")
	assert.NoError(t, err)
}

func TestGetProfitSharingDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetProfitSharingDetails(context.Background(), "", time.Now(), time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGetTotalProfitSharing(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetTotalProfitSharing(context.Background(), "SWAP")
	assert.NoError(t, err)
}

func TestGetUnrealizedProfitSharingDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetUnrealizedProfitSharingDetails(context.Background(), "SWAP")
	assert.NoError(t, err)
}

func TestSetFirstCopySettings(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.AmendCopySettings(context.Background(), &FirstCopySettings{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = ok.AmendCopySettings(context.Background(), &FirstCopySettings{
		InstrumentType:       "SWAP",
		UniqueCode:           "25CD5A80241D6FE6",
		CopyMgnMode:          "cross",
		CopyInstrumentIDType: "copy",
		CopyMode:             "ratio_copy",
		CopyRatio:            1,
		CopyTotalAmount:      500,
		SubPosCloseType:      "copy_close",
	})
	assert.NoError(t, err)
}

func TestAmendCopySettings(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.SetFirstCopySettings(context.Background(), &FirstCopySettings{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = ok.SetFirstCopySettings(context.Background(), &FirstCopySettings{
		InstrumentType:       "SWAP",
		UniqueCode:           "25CD5A80241D6FE6",
		CopyMgnMode:          "cross",
		CopyInstrumentIDType: "copy",
		CopyMode:             "ratio_copy",
		CopyRatio:            1,
		CopyTotalAmount:      500,
		SubPosCloseType:      "copy_close",
	})
	assert.NoError(t, err)
}

func TestStopCopying(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.StopCopying(context.Background(), &StopCopyingParameter{})
	require.ErrorIs(t, err, errNilArgument)
	_, err = ok.StopCopying(context.Background(), &StopCopyingParameter{
		InstrumentType:       "SWAP",
		UniqueCode:           "25CD5A80241D6FE6",
		SubPositionCloseType: "manual_close",
	})
	assert.NoError(t, err)
}

func TestGetCopySettings(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetCopySettings(context.Background(), "SWAP", "213E8C92DC61EFAC")
	assert.NoError(t, err)
}

func TestGetMultipleLeverages(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetMultipleLeverages(context.Background(), "", "213E8C92DC61EFAC", "")
	assert.True(t, !errors.Is(err, errMissingMarginMode), "expected %v, got %v", errMissingMarginMode, err)
	_, err = ok.GetMultipleLeverages(context.Background(), "isolated", "213E8C92DC61EFAC", "")
	assert.NoError(t, err)
}

func TestSetMultipleLeverages(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.SetMultipleLeverages(context.Background(), &SetLeveragesParam{
		MarginMode: "cross",
		Leverage:   5,
	})
	require.ErrorIs(t, err, errMissingInstrumentID)
	_, err = ok.SetMultipleLeverages(context.Background(), &SetLeveragesParam{
		MarginMode:   "cross",
		Leverage:     5,
		InstrumentID: "BTC-USDT-SWAP",
	})
	assert.NoError(t, err)
}

func TestGetMyLeadTraders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetMyLeadTraders(context.Background(), "SWAP")
	assert.NoError(t, err)
}

func TestGetHistoryLeadTraders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetHistoryLeadTraders(context.Background(), "", "", "", 10)
	assert.NoError(t, err)
}

func TestGetLeadTradersRanks(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetLeadTradersRanks(context.Background(), "SWAP", "pnl_ratio", "1", "", "", "", "", "", "", "", 10)
	assert.NoError(t, err)
}

func TestGetWeeklyTraderProfitAndLoss(t *testing.T) {
	t.Parallel()
	_, err := ok.GetWeeklyTraderProfitAndLoss(context.Background(), "", "213E8C92DC61EFAC")
	assert.NoError(t, err)
}

func TestGetDailyLeadTraderPNL(t *testing.T) {
	t.Parallel()
	_, err := ok.GetDailyLeadTraderPNL(context.Background(), "SWAP", "213E8C92DC61EFAC", "2")
	assert.NoError(t, err)
}

func TestGetLeadTraderStats(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeadTraderStats(context.Background(), "SWAP", "213E8C92DC61EFAC", "2")
	assert.NoError(t, err)
}

func TestGetLeadTraderCurrencyPreferences(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeadTraderCurrencyPreferences(context.Background(), "SWAP", "213E8C92DC61EFAC", "2")
	assert.NoError(t, err)
}

func TestGetLeadTraderCurrentLeadPositions(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeadTraderCurrentLeadPositions(context.Background(), "SPOT", "213E8C92DC61EFAC", "", "", 10)
	assert.Falsef(t, !errors.Is(err, asset.ErrNotSupported), "expected %v, got %v", asset.ErrNotSupported, err)
	_, err = ok.GetLeadTraderCurrentLeadPositions(context.Background(), "SWAP", "213E8C92DC61EFAC", "", "", 10)
	assert.NoError(t, err)
}

func TestGetLeadTraderLeadPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := ok.GetLeadTraderLeadPositionHistory(context.Background(), "SPOT", "213E8C92DC61EFAC", "", "", 10)
	require.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = ok.GetLeadTraderLeadPositionHistory(context.Background(), "SWAP", "213E8C92DC61EFAC", "", "", 10)
	assert.NoError(t, err)
}

func TestPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.PlaceSpreadOrder(context.Background(), &SpreadOrderParam{
		InstrumentID:  "BTC-USDT_BTC-USD-SWAP",
		SpreadID:      "1234",
		ClientOrderID: "12354123523",
		Side:          "buy",
		OrderType:     "limit",
		Size:          1,
		Price:         12345,
	})
	assert.NoError(t, err)
}

func TestCancelSpreadOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.CancelSpreadOrder(context.Background(), "12345", "")
	assert.NoError(t, err)
}

func TestWsCancelSpreadOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.WsCancelSpreadOrder("1234", "")
	assert.NoError(t, err)
}

func TestCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.CancelAllSpreadOrders(context.Background(), "123456")
	assert.NoError(t, err)
}

func TestWsCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.WsCancelAllSpreadOrders("BTC-USDT_BTC-USDT-SWAP")
	assert.NoError(t, err)
}

func TestAmendSpreadOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.AmendSpreadOrder(context.Background(), &AmendSpreadOrderParam{
		OrderID: "2510789768709120",
		NewSize: 2,
	})
	assert.NoError(t, err)
}

func TestWsAmandSpreadOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.WsAmandSpreadOrder(&AmendSpreadOrderParam{
		OrderID: "2510789768709120",
		NewSize: 2,
	})
	assert.NoError(t, err)
}

func TestGetSpreadOrderDetails(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetSpreadOrderDetails(context.Background(), "1234567", "")
	assert.NoError(t, err)
}

func TestGetActiveSpreadOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetActiveSpreadOrders(context.Background(), "", "post_only", "partially_filled", "", "", 10)
	assert.NoError(t, err)
}

func TestGetCompletedSpreadOrdersLast7Days(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetCompletedSpreadOrdersLast7Days(context.Background(), "", "limit", "canceled", "", "", time.Time{}, time.Time{}, 10)
	assert.NoError(t, err)
}

func TestGetSpreadTradesOfLast7Days(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetSpreadTradesOfLast7Days(context.Background(), "", "", "", "", "", time.Time{}, time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGetSpreads(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicSpreads(context.Background(), "", "", "", "")
	assert.NoError(t, err)
}

func TestGetSpreadOrderBooks(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicSpreadOrderBooks(context.Background(), "BTC-USDT_BTC-USDT-SWAP", 0)
	assert.NoError(t, err)
}

func TestGetSpreadTickers(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicSpreadTickers(context.Background(), "BTC-USDT_BTC-USDT-SWAP")
	assert.NoError(t, err)
}

func TestGetPublicSpreadTrades(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicSpreadTrades(context.Background(), "")
	assert.NoError(t, err)
}

func TestGetOptionsTickBands(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOptionsTickBands(context.Background(), "OPTION", "")
	assert.NoError(t, err)
}

func TestExtractIndexCandlestick(t *testing.T) {
	t.Parallel()
	data := IndexCandlestickSlices([][6]string{{"1597026383085", "3.721", "3.743", "3.677", "3.708", "1"}, {"1597026383085", "3.731", "3.799", "3.494", "3.72", "1"}})
	candlesticks, err := data.ExtractIndexCandlestick()
	assert.NoError(t, err)
	assert.False(t, len(candlesticks) != 2, "expected candles with length 2, got %d", len(candlesticks))
}

func TestGetHistoricIndexAndMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := ok.GetHistoricIndexCandlesticksHistory(context.Background(), "BTC-USD", time.Time{}, time.Time{}, kline.FiveMin, 10)
	assert.NoError(t, err)
	_, err = ok.GetMarkPriceCandlestickHistory(context.Background(), "BTC-USD-SWAP", time.Time{}, time.Time{}, kline.FiveMin, 10)
	assert.NoError(t, err)
}

func TestGetEconomicCanendarData(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetEconomicCalendarData(context.Background(), "", "", time.Now(), time.Time{}, 0)
	assert.NoError(t, err)
}

func TestGetDepositWithdrawalStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetDepositWithdrawalStatus(context.Background(), "1244", "", "", "", "")
	assert.NoError(t, err)
}

func TestGetPublicExchangeList(t *testing.T) {
	t.Parallel()
	_, err := ok.GetPublicExchangeList(context.Background())
	assert.NoError(t, err)
}

func TestWsPlaceSpreadOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok, canManipulateRealOrders)
	_, err := ok.WsPlaceSpreadOrder(&SpreadOrderParam{
		SpreadID:      "BTC-USDT_BTC-USDT-SWAP",
		ClientOrderID: "b15",
		Side:          "buy",
		OrderType:     "limit",
		Price:         2.15,
		Size:          2,
	})
	assert.NoError(t, err)
}

func TestGetInviteesDetail(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetInviteesDetail(context.Background(), "")
	assert.True(t, errors.Is(err, errUserIDRequired), "expected %v, got %v", errUserIDRequired, err)
	_, err = ok.GetInviteesDetail(context.Background(), "1234")
	assert.NoError(t, err)
}

func TestGetUserAffilateRebateInformation(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ok)
	_, err := ok.GetUserAffilateRebateInformation(context.Background(), "")
	require.ErrorIs(t, err, errInvalidAPIKey)
	_, err = ok.GetUserAffilateRebateInformation(context.Background(), "1234")
	assert.NoError(t, err)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := ok.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  currency.ETH.Item,
		Quote: currency.USDT.Item,
		Asset: asset.USDTMarginedFutures,
	})
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	usdSwapCode := currency.NewCode("USD-SWAP")
	resp, err := ok.GetOpenInterest(context.Background(), key.PairAsset{
		Base:  currency.BTC.Item,
		Quote: usdSwapCode.Item,
		Asset: asset.PerpetualSwap,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	cp1 := currency.NewPair(currency.DOGE, usdSwapCode)
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ok, asset.PerpetualSwap, cp1)
	resp, err = ok.GetOpenInterest(context.Background(),
		key.PairAsset{
			Base:  currency.BTC.Item,
			Quote: usdSwapCode.Item,
			Asset: asset.PerpetualSwap,
		},
		key.PairAsset{
			Base:  cp1.Base.Item,
			Quote: cp1.Quote.Item,
			Asset: asset.PerpetualSwap,
		},
	)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)

	resp, err = ok.GetOpenInterest(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}
