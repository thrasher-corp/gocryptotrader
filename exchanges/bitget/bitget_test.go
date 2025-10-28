package bitget

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	gws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// User-defined constants to aid testing
const (
	// Please supply your own keys here to do authenticated endpoint testing
	apiKey                  = ""
	apiSecret               = ""
	clientID                = "" // Passphrase made at API key creation
	canManipulateRealOrders = false
	testingInSandbox        = false
	// Needs to be set to a subaccount with deposit transfer permissions so that TestGetSubaccountDepositAddress doesn't fail
	deposSubaccID      = ""
	testSubaccountName = "GCTTESTA"
	testIP             = "14.203.57.50"
	testAddress        = "fake test address"
	// Test values used with live data, with the goal of never letting an order be executed
	testAmount = 0.001
	testPrice  = 1e10 - 3
	// Test values used with demo functionality, with the goal of lining up with the relatively strict currency limits present there
	testAmount2 = 0.003
	testPrice2  = 1667
)

// User-defined variables to aid testing
var (
	testCrypto  = currency.BTC   // Used for endpoints which don't support demo trading
	testCrpyto2 = currency.SBTC  // Used for endpoints which support demo trading
	testCrypto3 = currency.DOGE  // Used for endpoints which consume all available funds
	testFiat    = currency.USDT  // Used for endpoints which don't support demo trading
	testFiat2   = currency.SUSDT // Used for endpoints which support demo trading
	testPair    = currency.NewPair(testCrypto, testFiat)
	testPair2   = currency.NewPair(testCrpyto2, testFiat2)
)

// Developer-defined constants to aid testing
const (
	skipTestSubAccNotFound             = "appropriate sub-account (equals %v, not equals %v) not found, skipping"
	skipInsufficientAPIKeysFound       = "insufficient API keys found, skipping"
	skipInsufficientOrders             = "insufficient orders found, skipping"
	skipInsufficientRiskUnits          = "insufficient risk units found, skipping"
	skipInstitution                    = "this endpoint requires IDs tailored to an institution, so it can't be automatically tested, skipping"
	errAPIKeyLimitPartial              = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"40063","msg":"API exceeds the maximum limit added","requestTime":`
	errCurrentlyHoldingPositionPartial = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"45117","msg":"Currently holding positions or orders, the margin mode cannot be adjusted","requestTime":`
	meow                               = "meow"
	woof                               = "woof"
	neigh                              = "neigh"
)

// Developer-defined variables to aid testing
var (
	fakeCurrency = currency.NewCode("FAKECURRENCYNOT")
	fakePair     = currency.NewPair(fakeCurrency, currency.NewCode("REALMEOWMEOW"))

	errUnmarshalArray string
)

var e = &Exchange{}

func TestMain(m *testing.M) {
	e.SetDefaults()
	err := exchangeBaseHelper(e)
	if err != nil {
		log.Fatal(err)
	}
	if testingInSandbox {
		e.isDemoTrading = true
	}
	var dialer gws.Dialer
	err = e.Websocket.Conn.Dial(context.TODO(), &dialer, http.Header{})
	if err != nil {
		log.Fatal(err)
	}
	e.Websocket.Wg.Add(1)
	go e.wsReadData(e.Websocket.Conn)
	e.Websocket.Conn.SetupPingHandler(request.Unset, websocket.PingHandler{
		Message:     []byte(`ping`),
		MessageType: gws.TextMessage,
		Delay:       time.Second * 25,
	})
	switch json.Implementation {
	case "bytedance/sonic":
		errUnmarshalArray = "mismatched type with"
	case "encoding/json":
		errUnmarshalArray = "cannot unmarshal array"
	}
	os.Exit(m.Run())
}

func TestSetup(t *testing.T) {
	cfg, err := e.GetStandardConfig()
	assert.NoError(t, err)
	exch := &Exchange{}
	err = exch.Setup(nil)
	assert.ErrorIs(t, err, config.ErrExchangeConfigIsNil)
	exch.SetDefaults()
	err = exchangeBaseHelper(exch)
	require.NoError(t, err)
	cfg.Enabled = false
	err = exch.Setup(cfg)
	assert.NoError(t, err)
	assert.False(t, exch.IsEnabled())
	cfg.Enabled = true
	cfg.ProxyAddress = string(rune(0x7f))
	err = exch.Setup(cfg)
	assert.ErrorIs(t, err, exchange.ErrSettingProxyAddress)
	cfg.ProxyAddress = ""
	oldEP := exch.API.Endpoints
	exch.API.Endpoints = nil
	err = exch.Setup(cfg)
	assert.ErrorIs(t, err, exchange.ErrEndpointPathNotFound)
	exch.API.Endpoints = oldEP
	err = exch.Setup(cfg)
	assert.ErrorIs(t, err, websocket.ErrWebsocketAlreadyInitialised)
}

func TestWsConnect(t *testing.T) {
	exch := &Exchange{}
	exch.Websocket = sharedtestvalues.NewTestWebsocket()
	err := exch.Websocket.Disable()
	assert.ErrorIs(t, err, websocket.ErrAlreadyDisabled)
	err = exch.WsConnect()
	assert.ErrorIs(t, err, websocket.ErrWebsocketNotEnabled)
	exch.SetDefaults()
	err = exchangeBaseHelper(exch)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	exch.Verbose = true
	err = exch.WsConnect()
	assert.NoError(t, err)
}

func TestQueryAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := e.QueryAnnouncements(t.Context(), "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := e.QueryAnnouncements(t.Context(), "latest_news", time.Time{}, time.Time{}, 0, 10)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetTime(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetTime)
}

func TestGetTradeRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeRate(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetTradeRate(t.Context(), testPair, "")
	assert.ErrorIs(t, err, errBusinessTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetTradeRate(t.Context(), testPair, "spot")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAllTradeRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllTradeRates(t.Context(), "")
	assert.ErrorIs(t, err, errBusinessTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetAllTradeRates(t.Context(), "spot")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotTransactionRecords(t.Context(), currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSpotTransactionRecords(t.Context(), testFiat, time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetFuturesTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTransactionRecords(t.Context(), "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesTransactionRecords(t.Context(), "COIN-FUTURES", currency.Code{}, time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetMarginTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginTransactionRecords(t.Context(), "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMarginTransactionRecords(t.Context(), "crossed", testFiat, time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetP2PTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetP2PTransactionRecords(t.Context(), currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetP2PTransactionRecords(t.Context(), testFiat, time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetP2PMerchantList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetP2PMerchantList(t.Context(), nil, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetMerchantInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetMerchantInfo)
}

func TestGetMerchantP2POrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetMerchantP2POrders(t.Context(), time.Now().Add(time.Second), time.Now(), 0, 0, 0, 0, "", "", currency.Code{}, currency.Code{})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// Seems like it can't currently be properly tested due to not knowing any p2p order IDs
	_, err = e.GetMerchantP2POrders(t.Context(), time.Now().Add(-time.Hour*24*7), time.Now(), 5, 1, 0, 0, "", "", testCrypto, currency.Code{})
	assert.NoError(t, err)
}

func TestGetMerchantAdvertisementList(t *testing.T) {
	t.Parallel()
	_, err := e.GetMerchantAdvertisementList(t.Context(), time.Now().Add(time.Second), time.Now(), 0, 0, 0, 0, "", "", "", "", currency.Code{}, currency.Code{})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetMerchantAdvertisementList(t.Context(), time.Now().Add(-time.Hour*24*7), time.Now(), 5, 1<<62, 0, 0, "", "sell", "", "", testCrypto, currency.USD)
	assert.NoError(t, err)
}

func TestGetSpotWhaleNetFlow(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetSpotWhaleNetFlow, currency.Pair{}, testPair, currency.ErrCurrencyPairEmpty, true, false, false)
}

func TestGetFuturesActiveVolume(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesActiveVolume(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	e := new(Exchange) //nolint:govet // Intentional shadow // The endpoint intermittently returns "The data fetched by BTCUSDT is empty", while otherwise accepting that valid request. Mocking to avoid that flakiness.
	err = testexch.Setup(e)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/"+bitgetMix+bitgetMarket+bitgetTakerBuySell, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"code":"00000","msg":"success","requestTime":1758610429242,"data":[{"sellVolume":"61.87730903","buyVolume":"21.44177026","ts":"1758601500000"},{"sellVolume":"144.01881548","buyVolume":"112.84393002","ts":"1758601800000"},{"sellVolume":"43.49533167","buyVolume":"96.40228133","ts":"1758602100000"},{"sellVolume":"101.4114906","buyVolume":"213.27441876","ts":"1758602400000"},{"sellVolume":"31.2574504","buyVolume":"29.26275559","ts":"1758602700000"},{"sellVolume":"28.17317576","buyVolume":"43.50686056","ts":"1758603000000"},{"sellVolume":"26.3517612","buyVolume":"21.58492462","ts":"1758603300000"}]}`))
		assert.NoError(t, err)
	}))
	defer server.Close()
	err = e.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/")
	require.NoError(t, err)
	expectedData := []ActiveVolumeResp{
		{
			SellVolume: 61.87730903,
			BuyVolume:  21.44177026,
			Timestamp:  types.Time(time.UnixMilli(1758601500000)),
		},
		{
			SellVolume: 144.01881548,
			BuyVolume:  112.84393002,
			Timestamp:  types.Time(time.UnixMilli(1758601800000)),
		},
		{
			SellVolume: 43.49533167,
			BuyVolume:  96.40228133,
			Timestamp:  types.Time(time.UnixMilli(1758602100000)),
		},
		{
			SellVolume: 101.4114906,
			BuyVolume:  213.27441876,
			Timestamp:  types.Time(time.UnixMilli(1758602400000)),
		},
		{
			SellVolume: 31.2574504,
			BuyVolume:  29.26275559,
			Timestamp:  types.Time(time.UnixMilli(1758602700000)),
		},
		{
			SellVolume: 28.17317576,
			BuyVolume:  43.50686056,
			Timestamp:  types.Time(time.UnixMilli(1758603000000)),
		},
		{
			SellVolume: 26.3517612,
			BuyVolume:  21.58492462,
			Timestamp:  types.Time(time.UnixMilli(1758603300000)),
		},
	}
	result, err := e.GetFuturesActiveVolume(t.Context(), testPair, "")
	require.NoError(t, err)
	assert.Equal(t, expectedData, result)
}

func TestGetFuturesPositionRatios(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionRatios(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	e := new(Exchange) //nolint:govet // Intentional shadow // The endpoint intermittently returns "The data fetched by BTCUSDT is empty", while otherwise accepting that valid request. Mocking to avoid that flakiness.
	err = testexch.Setup(e)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/"+bitgetMix+bitgetMarket+bitgetPositionLongShort, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"code":"00000","msg":"success","requestTime":1758611552124,"data":[{"longPositionRatio":"0.5061","shortPositionRatio":"0.4939","longShortPositionRatio":"0.0102","ts":"1758602400000"},{"longPositionRatio":"0.5058","shortPositionRatio":"0.4942","longShortPositionRatio":"0.0102","ts":"1758602700000"},{"longPositionRatio":"0.5058","shortPositionRatio":"0.4942","longShortPositionRatio":"0.0102","ts":"1758603000000"},{"longPositionRatio":"0.5055","shortPositionRatio":"0.4945","longShortPositionRatio":"0.0102","ts":"1758603300000"}]}`))
		assert.NoError(t, err)
	}))
	defer server.Close()
	err = e.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/")
	require.NoError(t, err)
	expectedData := []PosRatFutureResp{
		{
			LongPositionRatio:      0.5061,
			ShortPositionRatio:     0.4939,
			LongShortPositionRatio: 0.0102,
			Timestamp:              types.Time(time.UnixMilli(1758602400000)),
		},
		{
			LongPositionRatio:      0.5058,
			ShortPositionRatio:     0.4942,
			LongShortPositionRatio: 0.0102,
			Timestamp:              types.Time(time.UnixMilli(1758602700000)),
		},
		{
			LongPositionRatio:      0.5058,
			ShortPositionRatio:     0.4942,
			LongShortPositionRatio: 0.0102,
			Timestamp:              types.Time(time.UnixMilli(1758603000000)),
		},
		{
			LongPositionRatio:      0.5055,
			ShortPositionRatio:     0.4945,
			LongShortPositionRatio: 0.0102,
			Timestamp:              types.Time(time.UnixMilli(1758603300000)),
		},
	}
	result, err := e.GetFuturesPositionRatios(t.Context(), testPair, "")
	require.NoError(t, err)
	assert.Equal(t, expectedData, result)
}

func TestGetMarginPositionRatios(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginPositionRatios(t.Context(), currency.Pair{}, "", currency.Code{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetMarginPositionRatios(t.Context(), testPair, "24h", currency.Code{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetMarginLoanGrowth(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarginLoanGrowth(t.Context(), currency.Pair{}, "", currency.Code{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetMarginLoanGrowth(t.Context(), testPair, "24h", currency.Code{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetIsolatedBorrowingRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedBorrowingRatio(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetIsolatedBorrowingRatio(t.Context(), testPair, "24h")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesRatios(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesRatios(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	e := new(Exchange) //nolint:govet // Intentional shadow // The endpoint intermittently returns "The data fetched by BTCUSDT is empty", while otherwise accepting that valid request. Mocking to avoid that flakiness.
	err = testexch.Setup(e)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/"+bitgetMix+bitgetMarket+bitgetLongShort, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"code":"00000","msg":"success","requestTime":1758611911133,"data":[{"longRatio":"0.7743","shortRatio":"0.2257","longShortRatio":"0.0343","ts":"1758602700000"},{"longRatio":"0.7743","shortRatio":"0.2257","longShortRatio":"0.0343","ts":"1758603000000"},{"longRatio":"0.7753","shortRatio":"0.2247","longShortRatio":"0.0345","ts":"1758603300000"},{"longRatio":"0.7753","shortRatio":"0.2247","longShortRatio":"0.0345","ts":"1758603600000"}]}`))
		assert.NoError(t, err)
	}))
	defer server.Close()
	err = e.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/")
	require.NoError(t, err)
	expectedData := []RatioResp{
		{
			LongRatio:      0.7743,
			ShortRatio:     0.2257,
			LongShortRatio: 0.0343,
			Timestamp:      types.Time(time.UnixMilli(1758602700000)),
		},
		{
			LongRatio:      0.7743,
			ShortRatio:     0.2257,
			LongShortRatio: 0.0343,
			Timestamp:      types.Time(time.UnixMilli(1758603000000)),
		},
		{
			LongRatio:      0.7753,
			ShortRatio:     0.2247,
			LongShortRatio: 0.0345,
			Timestamp:      types.Time(time.UnixMilli(1758603300000)),
		},
		{
			LongRatio:      0.7753,
			ShortRatio:     0.2247,
			LongShortRatio: 0.0345,
			Timestamp:      types.Time(time.UnixMilli(1758603600000)),
		},
	}
	result, err := e.GetFuturesRatios(t.Context(), testPair, "")
	require.NoError(t, err)
	assert.Equal(t, expectedData, result)
}

func TestGetSpotFundFlows(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotFundFlows(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetSpotFundFlows(t.Context(), testPair, "15m")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetTradeSupportSymbols(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetTradeSupportSymbols)
}

func TestGetSpotWhaleFundFlows(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetSpotWhaleFundFlows, currency.Pair{}, testPair, currency.ErrCurrencyPairEmpty, true, false, false)
}

func TestGetFuturesAccountRatios(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAccountRatios(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	e := new(Exchange) //nolint:govet // Intentional shadow // The endpoint intermittently returns "The data fetched by BTCUSDT is empty", while otherwise accepting that valid request. Mocking to avoid that flakiness.
	err = testexch.Setup(e)
	require.NoError(t, err)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/"+bitgetMix+bitgetMarket+bitgetAccountLongShort, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(`{"code":"00000","msg":"success","requestTime":1758612172170,"data":[{"longAccountRatio":"0.7724","shortAccountRatio":"0.2276","longShortAccountRatio":"0.0339","ts":"1758603000000"},{"longAccountRatio":"0.7734","shortAccountRatio":"0.2266","longShortAccountRatio":"0.0341","ts":"1758603300000"},{"longAccountRatio":"0.7735","shortAccountRatio":"0.2265","longShortAccountRatio":"0.0341","ts":"1758603600000"},{"longAccountRatio":"0.7737","shortAccountRatio":"0.2263","longShortAccountRatio":"0.0341","ts":"1758603900000"}]}`))
		assert.NoError(t, err)
	}))
	defer server.Close()
	err = e.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL+"/")
	require.NoError(t, err)
	expectedData := []AccountRatioResp{
		{
			LongAccountRatio:      0.7724,
			ShortAccountRatio:     0.2276,
			LongShortAccountRatio: 0.0339,
			Timestamp:             types.Time(time.UnixMilli(1758603000000)),
		},
		{
			LongAccountRatio:      0.7734,
			ShortAccountRatio:     0.2266,
			LongShortAccountRatio: 0.0341,
			Timestamp:             types.Time(time.UnixMilli(1758603300000)),
		},
		{
			LongAccountRatio:      0.7735,
			ShortAccountRatio:     0.2265,
			LongShortAccountRatio: 0.0341,
			Timestamp:             types.Time(time.UnixMilli(1758603600000)),
		},
		{
			LongAccountRatio:      0.7737,
			ShortAccountRatio:     0.2263,
			LongShortAccountRatio: 0.0341,
			Timestamp:             types.Time(time.UnixMilli(1758603900000)),
		},
	}
	result, err := e.GetFuturesAccountRatios(t.Context(), testPair, "")
	require.NoError(t, err)
	assert.Equal(t, expectedData, result)
}

func TestCreateVirtualSubaccounts(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.CreateVirtualSubaccounts, nil, []string{testSubaccountName}, errSubaccountEmpty, true, true, true)
}

func TestModifyVirtualSubaccount(t *testing.T) {
	t.Parallel()
	perms := []string{}
	_, err := e.ModifyVirtualSubaccount(t.Context(), "", "", perms)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = e.ModifyVirtualSubaccount(t.Context(), meow, "", perms)
	assert.ErrorIs(t, err, errNewStatusEmpty)
	_, err = e.ModifyVirtualSubaccount(t.Context(), meow, woof, perms)
	assert.ErrorIs(t, err, errNewPermsEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	tarID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	perms = append(perms, "read")
	resp, err := e.ModifyVirtualSubaccount(t.Context(), tarID, "normal", perms)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestCreateSubaccountAndAPIKey(t *testing.T) {
	t.Parallel()
	ipL := []string{}
	_, err := e.CreateSubaccountAndAPIKey(t.Context(), "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ipL = append(ipL, testIP)
	pL := []string{"read"}
	// Fails with error "subAccountList not empty" and I'm not sure why. The account I'm testing with is far off hitting the limit of 20 sub-accounts.
	// Now it's saying that parameter req cannot be empty, still no clue what that means
	// Now it's saying that parameter verification failed. Occasionally it says this in Chinese
	_, err = e.CreateSubaccountAndAPIKey(t.Context(), "OWMEOWME", "woofwoof123", "neighneighneighneigh", ipL, pL)
	assert.NoError(t, err)
}

func TestGetVirtualSubaccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetVirtualSubaccounts(t.Context(), 25, 1, "")
	assert.NoError(t, err)
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	ipL := []string{}
	_, err := e.CreateAPIKey(t.Context(), "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = e.CreateAPIKey(t.Context(), woof, "", "", ipL, ipL)
	assert.ErrorIs(t, err, errPassphraseEmpty)
	_, err = e.CreateAPIKey(t.Context(), woof, meow, "", ipL, ipL)
	assert.ErrorIs(t, err, errLabelEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	tarID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	ipL = append(ipL, testIP)
	pL := []string{"read"}
	_, err = e.CreateAPIKey(t.Context(), tarID, clientID, "neigh whinny", ipL, pL)
	if err != nil {
		if strings.Contains(err.Error(), errAPIKeyLimitPartial) {
			t.Log(err)
		} else {
			t.Error(err)
		}
	}
}

func TestModifyAPIKey(t *testing.T) {
	t.Parallel()
	ipL := []string{}
	_, err := e.ModifyAPIKey(t.Context(), "", "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errAPIKeyEmpty)
	_, err = e.ModifyAPIKey(t.Context(), "", "", "", woof, ipL, ipL)
	assert.ErrorIs(t, err, errPassphraseEmpty)
	_, err = e.ModifyAPIKey(t.Context(), "", meow, "", woof, ipL, ipL)
	assert.ErrorIs(t, err, errLabelEmpty)
	_, err = e.ModifyAPIKey(t.Context(), "", meow, "quack", woof, ipL, ipL)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	tarID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	resp, err := e.GetAPIKeys(t.Context(), tarID)
	assert.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientAPIKeysFound)
	}
	resp2, err := e.ModifyAPIKey(t.Context(), tarID, clientID, "oink", resp[0].SubaccountAPIKey, ipL, ipL)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2)
}

func TestGetAPIKeys(t *testing.T) {
	t.Parallel()
	var tarID string
	if sharedtestvalues.AreAPICredentialsSet(e) {
		tarID = subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	}
	testGetOneArg(t, e.GetAPIKeys, "", tarID, errSubaccountEmpty, true, true, true)
}

func TestGetFundingAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetFundingAssets, currency.Code{}, testCrypto, nil, false, true, true)
}

func TestGetBotAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetBotAccountAssets, "", "spot", nil, false, true, true)
}

func TestGetAssetOverview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetAssetOverview)
}

func TestGetConvertCoints(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetConvertCoins)
}

func TestGetQuotedPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetQuotedPrice(t.Context(), currency.Code{}, currency.Code{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetQuotedPrice(t.Context(), currency.NewCode(meow), currency.NewCode(woof), 0, 0)
	assert.ErrorIs(t, err, errFromToMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetQuotedPrice(t.Context(), testCrypto, testFiat, 0, 1)
	assert.NoError(t, err)
	resp, err := e.GetQuotedPrice(t.Context(), testCrypto, testFiat, 0.1, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetConvertHistory(t.Context(), time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetConvertHistory(t.Context(), time.Now().Add(-time.Hour*90*24), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetBGBConvertCoins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetBGBConvertCoins)
}

func TestConvertBGB(t *testing.T) {
	t.Parallel()
	// No matter what currency I use, this returns the error "currency does not support convert"; possibly a bad error message, with the true issue being lack of funds?
	testGetOneArg(t, e.ConvertBGB, nil, []currency.Code{testCrypto3}, currency.ErrCurrencyCodeEmpty, false, true, canManipulateRealOrders)
}

func TestGetBGBConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetBGBConvertHistory(t.Context(), 0, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetBGBConvertHistory(t.Context(), 0, 5, 1<<62, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetCoinInfo(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetCoinInfo, currency.Code{}, testCrypto, nil, true, false, false)
}

func TestGetSymbolInfo(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetSymbolInfo, currency.Pair{}, currency.Pair{}, nil, true, false, false)
}

func TestGetSpotVIPFeeRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetSpotVIPFeeRate)
}

func TestGetSpotTickerInformation(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetSpotTickerInformation, currency.Pair{}, testPair, nil, true, false, false)
}

func TestGetSpotMergeDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotMergeDepth(t.Context(), currency.Pair{}, "", "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetSpotMergeDepth(t.Context(), testPair, "scale3", "5")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetOrderbookDepth(t *testing.T) {
	t.Parallel()
	resp, err := e.GetOrderbookDepth(t.Context(), testPair, "step0", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotCandlestickData(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotCandlestickData(t.Context(), currency.Pair{}, "", time.Time{}, time.Time{}, 0, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetSpotCandlestickData(t.Context(), testPair, "", time.Time{}, time.Time{}, 0, false)
	assert.ErrorIs(t, err, errGranEmpty)
	_, err = e.GetSpotCandlestickData(t.Context(), testPair, woof, time.Time{}, time.Time{}, 5, true)
	assert.ErrorIs(t, err, errEndTimeEmpty)
	_, err = e.GetSpotCandlestickData(t.Context(), testPair, woof, time.Now().Add(time.Hour), time.Time{}, 0, false)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	_, err = e.GetSpotCandlestickData(t.Context(), testPair, "1min", time.Time{}, time.Time{}, 5, false)
	assert.NoError(t, err)
	resp, err := e.GetSpotCandlestickData(t.Context(), testPair, "1min", time.Time{}, time.Now(), 5, true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetRecentSpotFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentSpotFills(t.Context(), currency.Pair{}, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetRecentSpotFills(t.Context(), testPair, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotMarketTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotMarketTrades(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetSpotMarketTrades(t.Context(), testPair, time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := e.GetSpotMarketTrades(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestPlaceSpotOrder(t *testing.T) {
	t.Parallel()
	p := &PlaceSingleSpotOrderParams{}
	_, err := e.PlaceSpotOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair
	_, err = e.PlaceSpotOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errSideEmpty)
	p.Side = "sell"
	_, err = e.PlaceSpotOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	p.OrderType = "limit"
	_, err = e.PlaceSpotOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errStrategyEmpty)
	p.Strategy = "IOC"
	_, err = e.PlaceSpotOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	p.Price = testPrice
	_, err = e.PlaceSpotOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	p.ClientOrderID = "abc123"
	p.Amount = testAmount
	p.PresetTakeProfitPrice = testPrice - 1
	p.ExecuteTakeProfitPrice = testPrice - 2
	p.PresetStopLossPrice = testPrice + 1
	p.ExecuteStopLossPrice = testPrice + 2
	_, err = e.PlaceSpotOrder(t.Context(), p, true)
	assert.NoError(t, err)
	p.ClientOrderID = "321cba"
	p.TriggerPrice = testPrice / 10
	p.PresetTakeProfitPrice = 0
	p.ExecuteTakeProfitPrice = 0
	p.PresetStopLossPrice = 0
	p.ExecuteStopLossPrice = 0
	_, err = e.PlaceSpotOrder(t.Context(), p, false)
	assert.NoError(t, err)
}

func TestCancelAndPlaceSpotOrder(t *testing.T) {
	t.Parallel()
	p := &CancelAndPlaceSpotOrderParams{}
	_, err := e.CancelAndPlaceSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair
	_, err = e.CancelAndPlaceSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	p.OldClientOrderID = meow
	_, err = e.CancelAndPlaceSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, limits.ErrPriceBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.CancelAndPlaceSpotOrder(t.Context(), &CancelAndPlaceSpotOrderParams{Pair: testPair, OldClientOrderID: resp[0].ClientOrderID, NewClientOrderID: "a", Price: testPrice, Amount: testAmount, PresetTakeProfitPrice: testPrice - 1, ExecuteTakeProfitPrice: testPrice - 2, PresetStopLossPrice: testPrice + 1, ExecuteStopLossPrice: testPrice + 2, OrderID: uint64(resp[0].OrderID)})
	assert.NoError(t, err)
}

func TestBatchCancelAndPlaceSpotOrders(t *testing.T) {
	t.Parallel()
	var req []ReplaceSpotOrderParams
	_, err := e.BatchCancelAndPlaceSpotOrders(t.Context(), req)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	newPair, err := pairFromStringHelper(resp[0].Symbol)
	require.NoError(t, err)
	req = append(req, ReplaceSpotOrderParams{
		OrderID:          int64(resp[0].OrderID),
		OldClientOrderID: resp[0].ClientOrderID,
		Price:            testPrice,
		Amount:           testAmount,
		Pair:             newPair,
	})
	_, err = e.BatchCancelAndPlaceSpotOrders(t.Context(), req)
	assert.NoError(t, err)
}

func TestCancelSpotOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSpotOrderByID(t.Context(), currency.Pair{}, "", "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelSpotOrderByID(t.Context(), testPair, "", "", 0)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.CancelSpotOrderByID(t.Context(), testPair, "", resp[0].ClientOrderID, uint64(resp[0].OrderID))
	assert.NoError(t, err)
}

func TestBatchPlaceSpotOrders(t *testing.T) {
	t.Parallel()
	var req []PlaceSpotOrderParams
	_, err := e.BatchPlaceSpotOrders(t.Context(), currency.Pair{}, false, false, req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.BatchPlaceSpotOrders(t.Context(), testPair, false, false, req)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	req = append(req, PlaceSpotOrderParams{
		Side:      "sell",
		OrderType: "limit",
		Strategy:  "IOC",
		Price:     testPrice,
		Size:      testAmount,
		Pair:      testPair,
	})
	resp, err := e.BatchPlaceSpotOrders(t.Context(), testPair, true, true, req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestBatchCancelOrders(t *testing.T) {
	t.Parallel()
	var req []CancelSpotOrderParams
	_, err := e.BatchCancelOrders(t.Context(), currency.Pair{}, false, req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.BatchCancelOrders(t.Context(), testPair, false, req)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	pair, err := pairFromStringHelper(resp[0].Symbol)
	assert.NoError(t, err)
	req = append(req, CancelSpotOrderParams{
		OrderID:       uint64(resp[0].OrderID),
		ClientOrderID: resp[0].ClientOrderID,
		Pair:          pair,
	})
	resp2, err := e.BatchCancelOrders(t.Context(), testPair, true, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2)
}

func TestCancelOrderBySymbol(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.CancelOrdersBySymbol, currency.Pair{}, testPair, currency.ErrCurrencyPairEmpty, true, true, canManipulateRealOrders)
}

func TestGetSpotOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotOrderDetails(t.Context(), 0, "", 0)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSpotOrderDetails(t.Context(), 1, "a", time.Minute)
	assert.NoError(t, err)
}

func TestGetUnfilledOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetUnfilledOrders(t.Context(), currency.Pair{}, "", time.Now().Add(time.Hour), time.Time{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetUnfilledOrders(t.Context(), currency.Pair{}, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	assert.NoError(t, err)
}

func TestGetHistoricalSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalSpotOrders(t.Context(), currency.Pair{}, time.Now().Add(time.Hour), time.Time{}, 0, 0, 0, "", time.Minute)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetHistoricalSpotOrders(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 5, 1<<62, 0, "", time.Minute)
	assert.NoError(t, err)
}

func TestGetSpotFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotFills(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetSpotFills(t.Context(), testPair, time.Now().Add(time.Hour), time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSpotFills(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
}

func TestPlacePlanSpotOrder(t *testing.T) {
	t.Parallel()
	p := &PlaceSpotPlanOrderParams{}
	_, err := e.PlacePlanSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair
	_, err = e.PlacePlanSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, errSideEmpty)
	p.Side = woof
	_, err = e.PlacePlanSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	p.TriggerPrice = 1
	_, err = e.PlacePlanSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	p.OrderType = "limit"
	_, err = e.PlacePlanSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	p.OrderType = neigh
	_, err = e.PlacePlanSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	p.Amount = 1
	_, err = e.PlacePlanSpotOrder(t.Context(), p)
	assert.ErrorIs(t, err, errTriggerTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.PlacePlanSpotOrder(t.Context(), &PlaceSpotPlanOrderParams{Pair: testPair, Side: "sell", OrderType: "limit", TriggerType: "fill_price", ClientOrderID: "a", Strategy: "ioc", STPMode: "none", TriggerPrice: testPrice, ExecutePrice: testPrice, Amount: testAmount})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrentSpotPlanOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrentSpotPlanOrders(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestSpotGetPlanSubOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// This gets the error "the current plan order does not exist or has not been triggered" even when using a plan order that definitely exists and has definitely been triggered. Re-investigate later
	testGetOneArg(t, e.GetSpotPlanSubOrder, 0, 1, order.ErrOrderIDNotSet, true, true, true)
}

func TestGetSpotPlanOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotPlanOrderHistory(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetSpotPlanOrderHistory(t.Context(), testPair, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSpotPlanOrderHistory(t.Context(), testPair, time.Now().Add(-time.Hour*24*90), time.Now().Add(-time.Minute), 2, 1<<62)
	assert.NoError(t, err)
}

func TestBatchCancelSpotPlanOrders(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.BatchCancelSpotPlanOrders, nil, currency.Pairs{testPair}, nil, false, true, canManipulateRealOrders)
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// Not chucked into testGetNoArgs due to checking the presence of resp.Data, refactoring that generic for that would waste too many lines to do so just for this
	resp, err := e.GetAccountInfo(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetAccountAssets(t.Context(), testCrypto, "all")
	assert.NoError(t, err)
}

func TestGetSpotSubaccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetSpotSubaccountAssets)
}

func TestModifyDepositAccount(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyDepositAccount(t.Context(), "", currency.Code{})
	assert.ErrorIs(t, err, errAccountTypeEmpty)
	_, err = e.ModifyDepositAccount(t.Context(), meow, currency.Code{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.ModifyDepositAccount(t.Context(), "spot", testFiat)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAccountBills(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotAccountBills(t.Context(), currency.Code{}, "", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSpotAccountBills(t.Context(), testCrypto, "", "", time.Time{}, time.Time{}, 3, 1<<62)
	assert.NoError(t, err)
}

func TestTransferAsset(t *testing.T) {
	t.Parallel()
	_, err := e.TransferAsset(t.Context(), "", "", "", currency.Code{}, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = e.TransferAsset(t.Context(), meow, "", "", currency.Code{}, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errToTypeEmpty)
	_, err = e.TransferAsset(t.Context(), meow, woof, "", currency.Code{}, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errCurrencyAndPairEmpty)
	_, err = e.TransferAsset(t.Context(), meow, woof, "", currency.Code{}, testPair, 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.TransferAsset(t.Context(), "spot", "p2p", "a", testCrypto, testPair, testAmount)
	assert.NoError(t, err)
}

func TestGetTransferableCoinList(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransferableCoinList(t.Context(), "", "")
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = e.GetTransferableCoinList(t.Context(), meow, "")
	assert.ErrorIs(t, err, errToTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetTransferableCoinList(t.Context(), "spot", "p2p")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestSubaccountTransfer(t *testing.T) {
	t.Parallel()
	p := &SubaccountTransferParams{}
	_, err := e.SubaccountTransfer(t.Context(), p)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	p.FromType = meow
	_, err = e.SubaccountTransfer(t.Context(), p)
	assert.ErrorIs(t, err, errToTypeEmpty)
	p.ToType = woof
	_, err = e.SubaccountTransfer(t.Context(), p)
	assert.ErrorIs(t, err, errCurrencyAndPairEmpty)
	p.Cur = testCrypto
	_, err = e.SubaccountTransfer(t.Context(), p)
	assert.ErrorIs(t, err, errFromIDEmpty)
	p.FromID = neigh
	_, err = e.SubaccountTransfer(t.Context(), p)
	assert.ErrorIs(t, err, errToIDEmpty)
	p.ToID = "moo"
	_, err = e.SubaccountTransfer(t.Context(), p)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	fromID := subAccTestHelper(t, "", strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com")
	toID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	_, err = e.SubaccountTransfer(t.Context(), &SubaccountTransferParams{FromType: "spot", ToType: "p2p", ClientOrderID: "a", FromID: fromID, ToID: toID, Cur: testCrypto, Pair: testPair, Amount: testAmount})
	assert.NoError(t, err)
}

func TestWithdrawFunds(t *testing.T) {
	t.Parallel()
	p := &WithdrawFundsParams{}
	_, err := e.WithdrawFunds(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	p.Cur = testCrypto
	_, err = e.WithdrawFunds(t.Context(), p)
	assert.ErrorIs(t, err, errTransferTypeEmpty)
	p.TransferType = woof
	_, err = e.WithdrawFunds(t.Context(), p)
	assert.ErrorIs(t, err, errAddressEmpty)
	p.Address = neigh
	_, err = e.WithdrawFunds(t.Context(), p)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.WithdrawFunds(t.Context(), &WithdrawFundsParams{Cur: testCrypto, TransferType: "on_chain", Address: testAddress, Chain: testCrypto.String(), ClientOrderID: "a", Amount: testAmount})
	assert.NoError(t, err)
}

func TestGetSubaccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubaccountTransferRecord(t.Context(), currency.Code{}, "", "", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSubaccountTransferRecord(t.Context(), testCrypto, "", meow, "initiator", time.Time{}, time.Time{}, 3, 1<<62)
	assert.NoError(t, err)
}

func TestGetTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransferRecord(t.Context(), currency.Code{}, "", "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetTransferRecord(t.Context(), testCrypto, "", "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = e.GetTransferRecord(t.Context(), testCrypto, woof, "", time.Now().Add(time.Hour), time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetTransferRecord(t.Context(), testCrypto, "spot", meow, time.Time{}, time.Time{}, 3, 1<<62, 1)
	assert.NoError(t, err)
}

func TestSwitchBGBDeductionStatus(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.SwitchBGBDeductionStatus, false, false, nil, false, true, canManipulateRealOrders)
	testGetOneArg(t, e.SwitchBGBDeductionStatus, false, true, nil, false, true, canManipulateRealOrders)
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddressForCurrency(t.Context(), currency.Code{}, "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetDepositAddressForCurrency(t.Context(), testCrypto, "", 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSubaccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubaccountDepositAddress(t.Context(), "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = e.GetSubaccountDepositAddress(t.Context(), meow, "", currency.Code{}, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetSubaccountDepositAddress(t.Context(), deposSubaccID, "", testCrypto, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetBGBDeductionStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetBGBDeductionStatus)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := e.CancelWithdrawal(t.Context(), 0)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.WithdrawFunds(t.Context(), &WithdrawFundsParams{
		Cur:           testCrypto,
		TransferType:  "on_chain",
		Address:       testAddress,
		Chain:         testCrypto.String(),
		ClientOrderID: "a",
		Amount:        testAmount,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.OrderID)
	_, err = e.CancelWithdrawal(t.Context(), uint64(resp.OrderID))
	assert.NoError(t, err)
}

func TestGetSubaccountDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetSubaccountDepositRecords(t.Context(), "", currency.Code{}, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = e.GetSubaccountDepositRecords(t.Context(), meow, currency.Code{}, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	tarID := subAccTestHelper(t, "", "")
	_, err = e.GetSubaccountDepositRecords(t.Context(), tarID, currency.Code{}, 1<<62, 2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalRecords(t.Context(), currency.Code{}, "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetWithdrawalRecords(t.Context(), testCrypto, "", time.Now().Add(-time.Hour*24*90), time.Now(), 1<<62, 0, 5)
	assert.NoError(t, err)
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositRecords(t.Context(), currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDepositRecords(t.Context(), testCrypto, 0, 1<<62, 2, time.Now().Add(-time.Hour*24*90), time.Now())
	assert.NoError(t, err)
}

func TestGetFuturesVIPFeeRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetFuturesVIPFeeRate)
}

func TestGetInterestRateHistory(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetInterestRateHistory, currency.Code{}, testFiat, currency.ErrCurrencyCodeEmpty, true, false, false)
}

func TestGetInterestExchangeRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetInterestExchangeRate)
}

func TestGetDiscountRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetDiscountRate)
}

func TestGetFuturesMergeDepth(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesMergeDepth(t.Context(), currency.Pair{}, "", "", "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetFuturesMergeDepth(t.Context(), testPair, "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := e.GetFuturesMergeDepth(t.Context(), testPair, "USDT-FUTURES", "scale3", "5")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesTicker(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, e.GetFuturesTicker)
}

func TestGetAllFuturesTickers(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetAllFuturesTickers, "", "COIN-FUTURES", errProductTypeEmpty, true, false, false)
}

func TestGetRecentFuturesFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentFuturesFills(t.Context(), currency.Pair{}, "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetRecentFuturesFills(t.Context(), testPair, "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := e.GetRecentFuturesFills(t.Context(), testPair, "USDT-FUTURES", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesMarketTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesMarketTrades(t.Context(), currency.Pair{}, "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetFuturesMarketTrades(t.Context(), testPair, "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetFuturesMarketTrades(t.Context(), testPair, woof, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := e.GetFuturesMarketTrades(t.Context(), testPair, "USDT-FUTURES", 5, 1<<62, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesCandlestickData(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCandlestickData(t.Context(), currency.Pair{}, "", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetFuturesCandlestickData(t.Context(), testPair, "", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetFuturesCandlestickData(t.Context(), testPair, woof, "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errGranEmpty)
	_, err = e.GetFuturesCandlestickData(t.Context(), testPair, woof, neigh, "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := e.GetFuturesCandlestickData(t.Context(), testPair, "USDT-FUTURES", "1m", "", time.Time{}, time.Time{}, 5, CallModeNormal)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
	resp, err = e.GetFuturesCandlestickData(t.Context(), testPair, "COIN-FUTURES", "1m", "", time.Time{}, time.Time{}, 5, CallModeHistory)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
	resp, err = e.GetFuturesCandlestickData(t.Context(), testPair, "USDC-FUTURES", "1m", "", time.Time{}, time.Now(), 5, CallModeIndex)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
	resp, err = e.GetFuturesCandlestickData(t.Context(), testPair, "USDT-FUTURES", "1m", "", time.Time{}, time.Now(), 5, CallModeMark)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenPositions(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetOpenPositions(t.Context(), currency.NewPairWithDelimiter(meow, woof, ""), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetOpenPositions(t.Context(), testPair2, testFiat2.String()+"-FUTURES")
	assert.NoError(t, err)
}

func TestGetNextFundingTime(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, e.GetNextFundingTime)
}

func TestGetFuturesPrices(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, e.GetFuturesPrices)
}

func TestGetFundingHistorical(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingHistorical(t.Context(), currency.Pair{}, "", 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetFundingHistorical(t.Context(), testPair, "", 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := e.GetFundingHistorical(t.Context(), testPair, "USDT-FUTURES", 5, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFundingCurrent(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, e.GetFundingCurrent)
}

func TestGetContractConfig(t *testing.T) {
	t.Parallel()
	_, err := e.GetContractConfig(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetContractConfig(t.Context(), currency.Pair{}, prodTypes[0])
	assert.NoError(t, err)
}

func TestGetOneFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := e.GetOneFuturesAccount(t.Context(), currency.Pair{}, "", currency.Code{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetOneFuturesAccount(t.Context(), testPair, "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetOneFuturesAccount(t.Context(), testPair, woof, currency.Code{})
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetOneFuturesAccount(t.Context(), testPair, "USDT-FUTURES", testFiat)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAllFuturesAccounts(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetAllFuturesAccounts, "", "COIN-FUTURES", errProductTypeEmpty, true, true, true)
}

func TestGetFuturesSubaccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetFuturesSubaccountAssets, "", "COIN-FUTURES", errProductTypeEmpty, true, true, true)
}

func TestGetUSDTInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUSDTInterestHistory(t.Context(), testFiat, "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetUSDTInterestHistory(t.Context(), testFiat, woof, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// This endpoint persistently returns the error "Parameter verification failed" for no discernible reason
	_, err = e.GetUSDTInterestHistory(t.Context(), testFiat, "SUSDT-FUTURES", 1<<62, 2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetEstimatedOpenCount(t *testing.T) {
	t.Parallel()
	_, err := e.GetEstimatedOpenCount(t.Context(), currency.Pair{}, "", currency.Code{}, 0, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetEstimatedOpenCount(t.Context(), testPair, "", currency.Code{}, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetEstimatedOpenCount(t.Context(), testPair, woof, currency.Code{}, 0, 0, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = e.GetEstimatedOpenCount(t.Context(), testPair, woof, testFiat, 0, 0, 0)
	assert.ErrorIs(t, err, errOpenAmountEmpty)
	_, err = e.GetEstimatedOpenCount(t.Context(), testPair, woof, testFiat, 1, 0, 0)
	assert.ErrorIs(t, err, errOpenPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetEstimatedOpenCount(t.Context(), testPair, "USDT-FUTURES", testFiat, testPrice, testAmount, 20)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestSetIsolatedAutoMargin(t *testing.T) {
	t.Parallel()
	_, err := e.SetIsolatedAutoMargin(t.Context(), currency.Pair{}, false, currency.Code{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.SetIsolatedAutoMargin(t.Context(), testPair, false, currency.Code{}, "")
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = e.SetIsolatedAutoMargin(t.Context(), testPair, false, testFiat, "")
	assert.ErrorIs(t, err, errHoldSideEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.SetIsolatedAutoMargin(t.Context(), testPair, false, testFiat, "short")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestChangeLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeLeverage(t.Context(), currency.Pair{}, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.ChangeLeverage(t.Context(), testPair, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.ChangeLeverage(t.Context(), testPair, woof, "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = e.ChangeLeverage(t.Context(), testPair, woof, "", testFiat, 0)
	assert.ErrorIs(t, err, errLeverageEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.ChangeLeverage(t.Context(), testPair, "USDT-FUTURES", "", testFiat, 20)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestAdjustMargin(t *testing.T) {
	t.Parallel()
	err := e.AdjustMargin(t.Context(), currency.Pair{}, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	err = e.AdjustMargin(t.Context(), testPair2, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	err = e.AdjustMargin(t.Context(), testPair2, woof, "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	err = e.AdjustMargin(t.Context(), testPair2, woof, "", testFiat2, 0)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid)
	err = e.AdjustMargin(t.Context(), testPair2, woof, "", testFiat2, 1)
	assert.ErrorIs(t, err, errHoldSideEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	// This is getting the error "verification exception margin mode == FIXED", and I can't find a way to skirt around that
	// Now it's giving the error "insufficient amount of margin", which is a fine error to have, watch for random reversions
	// And back to the former error
	err = e.AdjustMargin(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "long", testFiat2, -testAmount)
	assert.NoError(t, err)
}

func TestSetUSDTAssetMode(t *testing.T) {
	t.Parallel()
	_, err := e.SetUSDTAssetMode(t.Context(), "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.SetUSDTAssetMode(t.Context(), meow, "")
	assert.ErrorIs(t, err, errAssetModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.SetUSDTAssetMode(t.Context(), "SUSDT-FUTURES", "single")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestChangeMarginMode(t *testing.T) {
	t.Parallel()
	_, err := e.ChangeMarginMode(t.Context(), currency.Pair{}, "", "", currency.Code{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.ChangeMarginMode(t.Context(), testPair2, "", "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.ChangeMarginMode(t.Context(), testPair2, woof, "", currency.Code{})
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = e.ChangeMarginMode(t.Context(), testPair2, woof, "", testFiat2)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.ChangeMarginMode(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "crossed", testFiat2)
	if err != nil {
		if strings.Contains(err.Error(), errCurrentlyHoldingPositionPartial) {
			t.Log(err)
		} else {
			t.Error(err)
		}
	}
}

func TestChangePositionMode(t *testing.T) {
	t.Parallel()
	_, err := e.ChangePositionMode(t.Context(), "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.ChangePositionMode(t.Context(), meow, "")
	assert.ErrorIs(t, err, errPositionModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.ChangePositionMode(t.Context(), testFiat2.String()+"-FUTURES", "hedge_mode")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesAccountBills(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAccountBills(t.Context(), "", "", "", currency.Code{}, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetFuturesAccountBills(t.Context(), meow, "", "", currency.Code{}, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesAccountBills(t.Context(), testFiat2.String()+"-FUTURES", "trans_from_exchange", "", testFiat2, 1<<62, 2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetPositionTier(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionTier(t.Context(), "", currency.Pair{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetPositionTier(t.Context(), meow, currency.Pair{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	resp, err := e.GetPositionTier(t.Context(), testFiat2.String()+"-FUTURES", testPair2)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSinglePosition(t *testing.T) {
	t.Parallel()
	_, err := e.GetSinglePosition(t.Context(), "", currency.Pair{}, currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetSinglePosition(t.Context(), meow, currency.Pair{}, currency.Code{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetSinglePosition(t.Context(), meow, testPair2, currency.Code{})
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSinglePosition(t.Context(), testFiat2.String()+"-FUTURES", testPair2, testFiat2)
	assert.NoError(t, err)
}

func TestGetAllPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllPositions(t.Context(), "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetAllPositions(t.Context(), meow, currency.Code{})
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetAllPositions(t.Context(), testFiat2.String()+"-FUTURES", testFiat2)
	assert.NoError(t, err)
}

func TestGetHistoricalPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalPositions(t.Context(), currency.Pair{}, "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeAndPairEmpty)
	_, err = e.GetHistoricalPositions(t.Context(), testPair2, "", 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetHistoricalPositions(t.Context(), testPair2, "", 1<<62, 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	p := &PlaceSingleFuturesOrderParams{}
	_, err := e.PlaceFuturesOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair2
	_, err = e.PlaceFuturesOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	p.ProductType = woof
	_, err = e.PlaceFuturesOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	p.MarginMode = neigh
	_, err = e.PlaceFuturesOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	p.MarginCoin = testFiat2
	_, err = e.PlaceFuturesOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errSideEmpty)
	p.Side = "oink"
	_, err = e.PlaceFuturesOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	p.OrderType = "limit"
	_, err = e.PlaceFuturesOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	p.Amount = testAmount2
	_, err = e.PlaceFuturesOrder(t.Context(), p, false)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.PlaceFuturesOrder(t.Context(), &PlaceSingleFuturesOrderParams{Pair: testPair2, ProductType: testFiat2.String() + "-FUTURES", MarginMode: "isolated", Side: "buy", TradeSide: "open", OrderType: "limit", Strategy: "GTC", ClientOrderID: "a", MarginCoin: testFiat2, StopSurplusPrice: testPrice2 + 1, StopLossPrice: testPrice2 - 1, Amount: testAmount2, Price: testPrice2, ReduceOnly: true}, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestPlaceReversal(t *testing.T) {
	t.Parallel()
	p := &PlaceReversalParams{}
	_, err := e.PlaceReversal(t.Context(), p, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair2
	_, err = e.PlaceReversal(t.Context(), p, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	p.MarginCoin = testFiat2
	_, err = e.PlaceReversal(t.Context(), p, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	p.ProductType = neigh
	_, err = e.PlaceReversal(t.Context(), p, false)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.PlaceReversal(t.Context(), &PlaceReversalParams{Pair: testPair2, MarginCoin: testFiat2, ProductType: testFiat2.String() + "-FUTURES", Side: "Buy", TradeSide: "Open", ClientOrderID: "a", Amount: 30}, true)
	assert.NoError(t, err)
}

func TestBatchPlaceFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.BatchPlaceFuturesOrders(t.Context(), currency.Pair{}, "", "", currency.Code{}, nil, false)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.BatchPlaceFuturesOrders(t.Context(), testPair2, "", "", currency.Code{}, nil, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.BatchPlaceFuturesOrders(t.Context(), testPair2, woof, "", currency.Code{}, nil, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = e.BatchPlaceFuturesOrders(t.Context(), testPair2, woof, "", testFiat2, nil, false)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	_, err = e.BatchPlaceFuturesOrders(t.Context(), testPair2, woof, neigh, testFiat2, nil, false)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orders := []PlaceFuturesOrderParams{
		{
			Size:      testAmount,
			Price:     testPrice,
			Side:      "Sell",
			TradeSide: "Open",
			OrderType: "limit",
			Strategy:  "FOK",
		},
	}
	_, err = e.BatchPlaceFuturesOrders(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "isolated", testFiat2, orders, true)
	assert.NoError(t, err)
}

func TestGetFuturesOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderDetails(t.Context(), currency.Pair{}, "", "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetFuturesOrderDetails(t.Context(), testPair2, "", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetFuturesOrderDetails(t.Context(), testPair2, woof, "", 0)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesOrderDetails(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "a", 1)
	assert.NoError(t, err)
}

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesFills(t.Context(), 0, 0, 0, currency.Pair{}, "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetFuturesFills(t.Context(), 0, 0, 0, currency.Pair{}, meow, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesFills(t.Context(), 0, 1<<62, 5, currency.Pair{}, testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesOrderFillHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderFillHistory(t.Context(), currency.Pair{}, "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetFuturesOrderFillHistory(t.Context(), currency.Pair{}, meow, 0, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// Keeps getting "Parameter verification failed" error and I can't figure out why
	// Now it's "Parameter verification failed 500"
	// And now it's "Parameter validation failed 510"
	resp, err := e.GetFuturesOrderFillHistory(t.Context(), currency.Pair{}, testFiat2.String()+"-FUTURES", 0, 1<<62, 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.FillList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetFuturesOrderFillHistory(t.Context(), currency.Pair{}, testFiat2.String()+"-FUTURES", resp.FillList[0].OrderID, 1<<62, 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetPendingFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetPendingFuturesOrders(t.Context(), 0, 0, 0, "", "", "", currency.Pair{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetPendingFuturesOrders(t.Context(), 0, 0, 0, "", meow, "", currency.Pair{}, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetPendingFuturesOrders(t.Context(), 0, 1<<62, 5, "", testFiat2.String()+"-FUTURES", "", testPair2, time.Now().Add(-time.Hour*24*90), time.Now())
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetPendingFuturesOrders(t.Context(), resp.EntrustedList[0].OrderID, 1<<62, 5, "", testFiat2.String()+"-FUTURES", "", testPair2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetHistoricalFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalFuturesOrders(t.Context(), 0, 0, 0, "", "", "", currency.Pair{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetHistoricalFuturesOrders(t.Context(), 0, 0, 0, "", meow, "", currency.Pair{}, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetHistoricalFuturesOrders(t.Context(), 0, 1<<62, 5, "", testFiat2.String()+"-FUTURES", "", testPair2, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetHistoricalFuturesOrders(t.Context(), resp.EntrustedList[0].OrderID, 1<<62, 5, "", testFiat2.String()+"-FUTURES", "", testPair2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesTriggerOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTriggerOrderByID(t.Context(), "", "", 0)
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = e.GetFuturesTriggerOrderByID(t.Context(), meow, "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetFuturesTriggerOrderByID(t.Context(), meow, woof, 0)
	assert.ErrorIs(t, err, errPlanOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 1<<62, 5, "", "normal_plan", "", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetFuturesTriggerOrderByID(t.Context(), "normal_plan", testFiat2.String()+"-FUTURES", resp.EntrustedList[0].OrderID)
	assert.NoError(t, err)
}

func TestPlaceTPSLFuturesOrder(t *testing.T) {
	t.Parallel()
	p := &PlaceTPSLFuturesOrderParams{}
	_, err := e.PlaceTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	p.MarginCoin = testFiat2
	_, err = e.PlaceTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	p.ProductType = woof
	_, err = e.PlaceTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair2
	_, err = e.PlaceTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	p.PlanType = neigh
	_, err = e.PlaceTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errHoldSideEmpty)
	p.HoldSide = "quack"
	_, err = e.PlaceTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	p.TriggerPrice = 1
	_, err = e.PlaceTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.PlaceTPSLFuturesOrder(t.Context(), &PlaceTPSLFuturesOrderParams{MarginCoin: testFiat2, ProductType: testFiat2.String() + "-FUTURES", PlanType: "profit_plan", HoldSide: "short", ClientOrderID: "a", Pair: testPair2, TriggerPrice: testPrice2 + 2, Amount: testAmount2})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestPlaceTPAndSLFuturesOrder(t *testing.T) {
	t.Parallel()
	p := &PlaceTPAndSLFuturesOrderParams{}
	_, err := e.PlaceTPAndSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	p.MarginCoin = testFiat2
	_, err = e.PlaceTPAndSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	p.ProductType = woof
	_, err = e.PlaceTPAndSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair2
	_, err = e.PlaceTPAndSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errTakeProfitTriggerPriceEmpty)
	p.TakeProfitTriggerPrice = 1
	_, err = e.PlaceTPAndSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errStopLossTriggerPriceEmpty)
	p.StopLossTriggerPrice = 1
	_, err = e.PlaceTPAndSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errHoldSideEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.PlaceTPAndSLFuturesOrder(t.Context(), &PlaceTPAndSLFuturesOrderParams{MarginCoin: testFiat2, ProductType: testFiat2.String() + "-FUTURES", TakeProfitTriggerType: "fill_price", StopLossTriggerType: "fill_price", HoldSide: "short", Pair: testPair2, TakeProfitTriggerPrice: 2*testPrice2 + 1, TakeProfitExecutePrice: 2 * testPrice2, StopLossTriggerPrice: testPrice2 - 1, StopLossExecutePrice: testPrice2})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestPlaceTriggerFuturesOrder(t *testing.T) {
	t.Parallel()
	p := &PlaceTriggerFuturesOrderParams{}
	_, err := e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	p.PlanType = meow
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair2
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	p.ProductType = woof
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	p.MarginMode = neigh
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	p.MarginCoin = testFiat2
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errTriggerTypeEmpty)
	p.TriggerType = "oink"
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errSideEmpty)
	p.Side = "quack"
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	p.OrderType = "cluck"
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	p.Amount = 1
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errExecutePriceEmpty)
	p.ExecutePrice = 1
	_, err = e.PlaceTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	// This returns the error "The parameter does not meet the specification d delegateType is error". The documentation doesn't mention that parameter anywhere, nothing seems similar to it, and attempts to send that parameter with various values, or to tweak other parameters, yielded no difference
	resp, err := e.PlaceTriggerFuturesOrder(t.Context(), &PlaceTriggerFuturesOrderParams{PlanType: "normal_plan", ProductType: testFiat2.String() + "-FUTURES", MarginMode: "isolated", TriggerType: "mark_price", Side: "Sell", OrderType: "limit", ClientOrderID: "a", TakeProfitTriggerType: "fill_price", StopLossTriggerType: "fill_price", Pair: testPair2, MarginCoin: testFiat2, Amount: testAmount2 * 1000, ExecutePrice: testPrice2 + 2, TriggerPrice: testPrice2 + 1, TakeProfitTriggerPrice: testPrice2 + 3, TakeProfitExecutePrice: testPrice2 - 2, StopLossTriggerPrice: testPrice2 - 1, StopLossExecutePrice: testPrice2 / 2, ReduceOnly: true})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyTPSLFuturesOrder(t *testing.T) {
	t.Parallel()
	p := &ModifyTPSLFuturesOrderParams{}
	_, err := e.ModifyTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	p.OrderID = 1
	_, err = e.ModifyTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	p.MarginCoin = testFiat2
	_, err = e.ModifyTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	p.ProductType = meow
	_, err = e.ModifyTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair2
	_, err = e.ModifyTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	p.TriggerPrice = 1
	_, err = e.ModifyTPSLFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.ModifyTPSLFuturesOrder(t.Context(), &ModifyTPSLFuturesOrderParams{OrderID: 1, ClientOrderID: "a", ProductType: testFiat2.String() + "-FUTURES", MarginCoin: testFiat2, Pair: testPair2, TriggerPrice: testPrice2 - 1, ExecutePrice: testPrice2 + 2, Amount: testAmount2, RangeRate: 0.1})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyTriggerFuturesOrder(t *testing.T) {
	t.Parallel()
	p := &ModifyTriggerFuturesOrderParams{}
	_, err := e.ModifyTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	p.OrderID = 1
	_, err = e.ModifyTriggerFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	t.Skip("TODO: Finish once PlaceTriggerFuturesOrder is fixed")
}

func TestGetPendingTriggerFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetPendingTriggerFuturesOrders(t.Context(), 0, 0, 0, "", "", "", currency.Pair{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = e.GetPendingTriggerFuturesOrders(t.Context(), 0, 0, 0, "", meow, "", currency.Pair{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetPendingTriggerFuturesOrders(t.Context(), 0, 0, 0, "", meow, woof, currency.Pair{}, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetPendingTriggerFuturesOrders(t.Context(), 0, 1<<62, 5, "", "profit_loss", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetPendingTriggerFuturesOrders(t.Context(), resp.EntrustedList[0].OrderID, 1<<62, 5, "", "profit_loss", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetHistoricalTriggerFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 0, 0, "", "", "", "", testPair2, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = e.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 0, 0, "", meow, "", "", testPair2, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 0, 0, "", meow, "", woof, testPair2, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 1<<62, 5, "", "normal_plan", "", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetHistoricalTriggerFuturesOrders(t.Context(), resp.EntrustedList[0].OrderID, 1<<62, 5, "", "normal_plan", "", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetSupportedCurrencies(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, e.GetSupportedCurrencies)
}

func TestGetCrossBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossBorrowHistory(t.Context(), 0, 0, 0, currency.Code{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossBorrowHistory(t.Context(), 1, 2, 1<<62, testFiat, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossRepayHistory(t.Context(), 0, 0, 0, currency.Code{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossRepayHistory(t.Context(), 1, 2, 1<<62, testFiat, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossInterestHistory(t.Context(), currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossInterestHistory(t.Context(), testFiat, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetCrossLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossLiquidationHistory(t.Context(), time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossLiquidationHistory(t.Context(), time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetCrossFinancialHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossFinancialHistory(t.Context(), "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossFinancialHistory(t.Context(), "", testFiat, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetCrossAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetCrossAccountAssets, currency.Code{}, testFiat, nil, false, true, true)
}

func TestCrossBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.CrossBorrow(t.Context(), currency.Code{}, "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CrossBorrow(t.Context(), testFiat, "", 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CrossBorrow(t.Context(), testFiat, "a", testAmount)
	assert.NoError(t, err)
}

func TestCrossRepay(t *testing.T) {
	t.Parallel()
	_, err := e.CrossRepay(t.Context(), currency.Code{}, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CrossRepay(t.Context(), testFiat, 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CrossRepay(t.Context(), testFiat, testAmount)
	assert.NoError(t, err)
}

func TestGetCrossRiskRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetCrossRiskRate)
}

func TestGetCrossMaxBorrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetCrossMaxBorrowable, currency.Code{}, testFiat, currency.ErrCurrencyCodeEmpty, false, true, true)
}

func TestGetCrossMaxTransferable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetCrossMaxTransferable, currency.Code{}, testFiat, currency.ErrCurrencyCodeEmpty, false, true, true)
}

func TestGetCrossInterestRateAndMaxBorrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetCrossInterestRateAndMaxBorrowable, currency.Code{}, testFiat, currency.ErrCurrencyCodeEmpty, false, true, true)
}

func TestGetCrossTierConfiguration(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetCrossTierConfiguration, currency.Code{}, testFiat, currency.ErrCurrencyCodeEmpty, false, true, true)
}

func TestCrossFlashRepay(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.CrossFlashRepay, currency.Code{}, testFiat, nil, false, true, canManipulateRealOrders)
}

func TestGetCrossFlashRepayResult(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossFlashRepayResult(t.Context(), nil)
	assert.ErrorIs(t, err, errIDListEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	// This must be done, as this is the only way to get a repayment ID
	resp, err := e.CrossFlashRepay(t.Context(), testFiat)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = e.GetCrossFlashRepayResult(t.Context(), []int64{resp.RepayID})
	assert.NoError(t, err)
}

func TestPlaceCrossOrder(t *testing.T) {
	t.Parallel()
	p := &PlaceMarginOrderParams{}
	_, err := e.PlaceCrossOrder(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair
	_, err = e.PlaceCrossOrder(t.Context(), p)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	p.OrderType = woof
	_, err = e.PlaceCrossOrder(t.Context(), p)
	assert.ErrorIs(t, err, errLoanTypeEmpty)
	p.LoanType = neigh
	_, err = e.PlaceCrossOrder(t.Context(), p)
	assert.ErrorIs(t, err, errStrategyEmpty)
	p.Strategy = "oink"
	_, err = e.PlaceCrossOrder(t.Context(), p)
	assert.ErrorIs(t, err, errSideEmpty)
	p.Side = "quack"
	_, err = e.PlaceCrossOrder(t.Context(), p)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.PlaceCrossOrder(t.Context(), &PlaceMarginOrderParams{Pair: testPair, OrderType: "limit", LoanType: "normal", Strategy: "GTC", Side: "Sell", Price: testPrice, BaseAmount: testAmount, QuoteAmount: testAmount * testPrice})
	assert.NoError(t, err)
}

func TestBatchPlaceCrossOrders(t *testing.T) {
	t.Parallel()
	_, err := e.BatchPlaceCrossOrders(t.Context(), currency.Pair{}, nil)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.BatchPlaceCrossOrders(t.Context(), testPair, nil)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orders := []MarginOrderData{
		{
			BaseAmount: testAmount2,
			Price:      testPrice2,
			OrderType:  "limit",
			Strategy:   "GTC",
			LoanType:   "normal",
			Side:       "Buy",
		},
	}
	_, err = e.BatchPlaceCrossOrders(t.Context(), testPair, orders)
	assert.NoError(t, err)
}

func TestGetCrossOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossOpenOrders(t.Context(), currency.Pair{}, "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetCrossOpenOrders(t.Context(), testPair, "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossOpenOrders(t.Context(), testPair, "", 1, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossHistoricalorders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossHistoricalOrders(t.Context(), currency.Pair{}, "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetCrossHistoricalOrders(t.Context(), testPair, "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetCrossHistoricalOrders(t.Context(), testPair, "", "", 0, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetCrossHistoricalOrders(t.Context(), testPair, "", "", resp.OrderList[0].OrderID, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossOrderFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossOrderFills(t.Context(), currency.Pair{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetCrossOrderFills(t.Context(), testPair, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetCrossOrderFills(t.Context(), testPair, 0, 1<<62, 5, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Fills) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetCrossOrderFills(t.Context(), testPair, resp.Fills[0].OrderID, 1<<62, 5, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCrossLiquidationOrders(t.Context(), "", "", "", currency.Pair{}, time.Now().Add(time.Hour), time.Time{}, 5, 1<<62)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetCrossLiquidationOrders(t.Context(), "swap", "", "", currency.Pair{}, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedRepayHistory(t.Context(), currency.Pair{}, currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetIsolatedRepayHistory(t.Context(), testPair, currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetIsolatedRepayHistory(t.Context(), testPair, currency.Code{}, 0, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.ResultList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetIsolatedRepayHistory(t.Context(), testPair, testFiat, resp.ResultList[0].RepayID, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedBorrowHistory(t.Context(), currency.Pair{}, currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetIsolatedBorrowHistory(t.Context(), testPair, currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedBorrowHistory(t.Context(), testPair, currency.Code{}, 0, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedInterestHistory(t.Context(), currency.Pair{}, currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetIsolatedInterestHistory(t.Context(), testPair, currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedInterestHistory(t.Context(), testPair, testFiat, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedLiquidationHistory(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetIsolatedLiquidationHistory(t.Context(), testPair, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedLiquidationHistory(t.Context(), testPair, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedFinancialHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedFinancialHistory(t.Context(), currency.Pair{}, "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetIsolatedFinancialHistory(t.Context(), testPair, "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedFinancialHistory(t.Context(), testPair, "", testCrypto, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetIsolatedAccountAssets, currency.Pair{}, currency.Pair{}, nil, false, true, true)
}

func TestIsolatedBorrow(t *testing.T) {
	t.Parallel()
	_, err := e.IsolatedBorrow(t.Context(), currency.Pair{}, currency.Code{}, "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.IsolatedBorrow(t.Context(), testPair, currency.Code{}, "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.IsolatedBorrow(t.Context(), testPair, testFiat, "", 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.IsolatedBorrow(t.Context(), testPair, testFiat, "", testAmount)
	assert.NoError(t, err)
}

func TestIsolatedRepay(t *testing.T) {
	t.Parallel()
	_, err := e.IsolatedRepay(t.Context(), 0, currency.Code{}, currency.Pair{}, "")
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.IsolatedRepay(t.Context(), 1, currency.Code{}, currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.IsolatedRepay(t.Context(), 1, testFiat, currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.IsolatedRepay(t.Context(), testAmount, testFiat, testPair, "")
	assert.NoError(t, err)
}

func TestGetIsolatedRiskRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetIsolatedRiskRate(t.Context(), currency.Pair{}, 1, 5)
	assert.NoError(t, err)
}

func TestGetIsolatedInterestRateAndMaxBorrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetIsolatedInterestRateAndMaxBorrowable, currency.Pair{}, testPair, currency.ErrCurrencyPairEmpty, false, true, true)
}

func TestGetIsolatedTierConfiguration(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetIsolatedTierConfiguration, currency.Pair{}, testPair, currency.ErrCurrencyPairEmpty, false, true, true)
}

func TestGetIsolatedMaxborrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetIsolatedMaxBorrowable, currency.Pair{}, testPair, currency.ErrCurrencyPairEmpty, false, true, true)
}

func TestGetIsolatedMaxTransferable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetIsolatedMaxTransferable, currency.Pair{}, testPair, currency.ErrCurrencyPairEmpty, false, true, true)
}

func TestIsolatedFlashRepay(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.IsolatedFlashRepay, nil, currency.Pairs{testPair}, nil, false, true, canManipulateRealOrders)
}

func TestGetIsolatedFlashRepayResult(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedFlashRepayResult(t.Context(), nil)
	assert.ErrorIs(t, err, errIDListEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	// This must be done, as this is the only way to get a repayment ID
	resp, err := e.IsolatedFlashRepay(t.Context(), currency.Pairs{testPair})
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = e.GetIsolatedFlashRepayResult(t.Context(), []int64{resp[0].RepayID})
	assert.NoError(t, err)
}

func TestPlaceIsolatedOrder(t *testing.T) {
	t.Parallel()
	p := &PlaceMarginOrderParams{}
	_, err := e.PlaceIsolatedOrder(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair
	_, err = e.PlaceIsolatedOrder(t.Context(), p)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	p.OrderType = meow
	_, err = e.PlaceIsolatedOrder(t.Context(), p)
	assert.ErrorIs(t, err, errLoanTypeEmpty)
	p.LoanType = woof
	_, err = e.PlaceIsolatedOrder(t.Context(), p)
	assert.ErrorIs(t, err, errStrategyEmpty)
	p.Strategy = neigh
	_, err = e.PlaceIsolatedOrder(t.Context(), p)
	assert.ErrorIs(t, err, errSideEmpty)
	p.Side = "quack"
	_, err = e.PlaceIsolatedOrder(t.Context(), p)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.PlaceIsolatedOrder(t.Context(), &PlaceMarginOrderParams{Pair: testPair, OrderType: "limit", LoanType: "normal", Strategy: "GTC", Side: "Sell", Price: testPrice, BaseAmount: testAmount, QuoteAmount: testAmount * testPrice})
	assert.NoError(t, err)
}

func TestBatchPlaceIsolatedOrders(t *testing.T) {
	t.Parallel()
	_, err := e.BatchPlaceIsolatedOrders(t.Context(), currency.Pair{}, nil)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.BatchPlaceIsolatedOrders(t.Context(), testPair, nil)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orders := []MarginOrderData{
		{
			BaseAmount: testAmount2,
			Price:      testPrice2,
			OrderType:  "limit",
			Strategy:   "GTC",
			LoanType:   "normal",
			Side:       "Buy",
		},
	}
	_, err = e.BatchPlaceIsolatedOrders(t.Context(), testPair, orders)
	assert.NoError(t, err)
}

func TestGetIsolatedOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedOpenOrders(t.Context(), currency.Pair{}, "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetIsolatedOpenOrders(t.Context(), testPair, "", 0, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedOpenOrders(t.Context(), testPair, "", 1, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedHistoricalOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedHistoricalOrders(t.Context(), currency.Pair{}, "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetIsolatedHistoricalOrders(t.Context(), testPair, "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetIsolatedHistoricalOrders(t.Context(), testPair, "", "", 0, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetIsolatedHistoricalOrders(t.Context(), testPair, "", "", resp.OrderList[0].OrderID, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedOrderFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedOrderFills(t.Context(), currency.Pair{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetIsolatedOrderFills(t.Context(), testPair, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetIsolatedOrderFills(t.Context(), testPair, 0, 1<<62, 5, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Fills) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetIsolatedOrderFills(t.Context(), testPair, resp.Fills[0].OrderID, 1<<62, 5, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetIsolatedLiquidationOrders(t.Context(), "", "", "", currency.Pair{}, time.Now().Add(time.Hour), time.Time{}, 5, 1<<62)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetIsolatedLiquidationOrders(t.Context(), "place_order", "", "", currency.Pair{}, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSavingsProductList(t *testing.T) {
	t.Parallel()
	_, err := e.GetSavingsProductList(t.Context(), currency.Code{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSavingsProductList(t.Context(), testCrypto, "")
	assert.NoError(t, err)
}

func TestGetSavingsBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetSavingsBalance)
}

func TestGetSavingsAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetSavingsAssets(t.Context(), "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSavingsAssets(t.Context(), "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSavingsRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetSavingsRecords(t.Context(), currency.Code{}, "", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSavingsRecords(t.Context(), currency.Code{}, "", "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSavingsSubscriptionDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetSavingsSubscriptionDetail(t.Context(), 0, "")
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = e.GetSavingsSubscriptionDetail(t.Context(), 1, "")
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetSavingsProductList(t.Context(), testCrypto, "")
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = e.GetSavingsSubscriptionDetail(t.Context(), resp[0].ProductID, resp[0].PeriodType)
	assert.NoError(t, err)
}

func TestSubscribeSavings(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeSavings(t.Context(), 0, "", 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = e.SubscribeSavings(t.Context(), 1, "", 0)
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	_, err = e.SubscribeSavings(t.Context(), 1, meow, 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetSavingsProductList(t.Context(), testCrypto, "")
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	resp2, err := e.GetSavingsSubscriptionDetail(t.Context(), resp[0].ProductID, resp[0].PeriodType)
	require.NoError(t, err)
	require.NotEmpty(t, resp2)
	_, err = e.SubscribeSavings(t.Context(), resp[0].ProductID, resp[0].PeriodType, resp2.SingleMinAmount.Float64())
	assert.NoError(t, err)
}

func TestGetSavingsSubscriptionResult(t *testing.T) {
	t.Parallel()
	_, err := e.GetSavingsSubscriptionResult(t.Context(), 0, "")
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = e.GetSavingsSubscriptionResult(t.Context(), 1, "")
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetSavingsRecords(t.Context(), currency.Code{}, "", "", time.Time{}, time.Time{}, 100, 0)
	require.NoError(t, err)
	require.NotEmpty(t, resp.ResultList)
	tarID := -1
	for x := range resp.ResultList {
		if resp.ResultList[x].OrderType == "subscribe" {
			tarID = x
			break
		}
	}
	if tarID == -1 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetSavingsSubscriptionResult(t.Context(), resp.ResultList[tarID].OrderID, resp.ResultList[tarID].ProductType)
	assert.NoError(t, err)
}

func TestRedeemSavings(t *testing.T) {
	t.Parallel()
	_, err := e.RedeemSavings(t.Context(), 0, 0, "", 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = e.RedeemSavings(t.Context(), 1, 0, "", 0)
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	_, err = e.RedeemSavings(t.Context(), 1, 0, meow, 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	var pagination uint64
	var tarProd uint64
	var tarOrd uint64
	var tarPeriod string
	for {
		resp, err := e.GetSavingsAssets(t.Context(), "", time.Time{}, time.Time{}, 100, pagination)
		require.NoError(t, err)
		require.NotEmpty(t, resp)
		for i := range resp.ResultList {
			if resp.ResultList[i].AllowRedemption && resp.ResultList[i].Status != "in_redemption" {
				tarProd = resp.ResultList[i].ProductID
				tarOrd = resp.ResultList[i].OrderID
				tarPeriod = resp.ResultList[i].PeriodType
				break
			}
		}
		if tarProd != 0 || uint64(resp.EndID) == pagination || resp.EndID == 0 {
			break
		}
		pagination = uint64(resp.EndID)
	}
	if tarProd == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.RedeemSavings(t.Context(), tarProd, tarOrd, tarPeriod, testAmount)
	assert.NoError(t, err)
}

func TestGetSavingsRedemptionResult(t *testing.T) {
	t.Parallel()
	_, err := e.GetSavingsRedemptionResult(t.Context(), 0, "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.GetSavingsRedemptionResult(t.Context(), 1, "")
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetSavingsRecords(t.Context(), currency.Code{}, "", "", time.Time{}, time.Time{}, 100, 0)
	require.NoError(t, err)
	require.NotEmpty(t, resp.ResultList)
	tarID := -1
	for x := range resp.ResultList {
		if resp.ResultList[x].OrderType == "redeem" {
			tarID = x
			break
		}
	}
	if tarID == -1 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetSavingsRedemptionResult(t.Context(), resp.ResultList[tarID].OrderID, resp.ResultList[tarID].ProductType)
	assert.NoError(t, err)
}

func TestGetEarnAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetEarnAccountAssets, currency.Code{}, currency.Code{}, nil, false, true, true)
}

func TestGetSharkFinProducts(t *testing.T) {
	t.Parallel()
	_, err := e.GetSharkFinProducts(t.Context(), currency.Code{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSharkFinProducts(t.Context(), testCrypto, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSharkFinBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetSharkFinBalance)
}

func TestGetSharkFinAssets(t *testing.T) {
	t.Parallel()
	_, err := e.GetSharkFinAssets(t.Context(), "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSharkFinAssets(t.Context(), "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSharkFinRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetSharkFinRecords(t.Context(), currency.Code{}, "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetSharkFinRecords(t.Context(), currency.Code{}, "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSharkFinSubscriptionDetail(t *testing.T) {
	t.Parallel()
	_, err := e.GetSharkFinSubscriptionDetail(t.Context(), 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetSharkFinProducts(t.Context(), testCrypto, 5, 1<<62)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = e.GetSharkFinSubscriptionDetail(t.Context(), resp.ResultList[0].ProductID)
	assert.NoError(t, err)
}

func TestSubscribeSharkFin(t *testing.T) {
	t.Parallel()
	_, err := e.SubscribeSharkFin(t.Context(), 0, 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = e.SubscribeSharkFin(t.Context(), 1, 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetSharkFinProducts(t.Context(), testCrypto, 5, 1<<62)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = e.SubscribeSharkFin(t.Context(), resp.ResultList[0].ProductID, resp.ResultList[0].MinimumAmount.Float64())
	assert.NoError(t, err)
}

func TestGetSharkFinSubscriptionResult(t *testing.T) {
	t.Parallel()
	_, err := e.GetSharkFinSubscriptionResult(t.Context(), 0)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetSharkFinRecords(t.Context(), currency.Code{}, "", time.Time{}, time.Time{}, 100, 0)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	tarID := -1
	for x := range resp {
		if resp[x].Type == "subscribe" {
			tarID = x
			break
		}
	}
	if tarID == -1 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetSharkFinSubscriptionResult(t.Context(), resp[tarID].OrderID)
	assert.NoError(t, err)
}

func TestGetLoanCurrencyList(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetLoanCurrencyList, currency.Code{}, testFiat, currency.ErrCurrencyCodeEmpty, false, true, true)
}

func TestGetEstimatedInterestAndBorrowable(t *testing.T) {
	t.Parallel()
	_, err := e.GetEstimatedInterestAndBorrowable(t.Context(), currency.Code{}, currency.Code{}, "", 0)
	assert.ErrorIs(t, err, errLoanCoinEmpty)
	_, err = e.GetEstimatedInterestAndBorrowable(t.Context(), testCrypto, currency.Code{}, "", 0)
	assert.ErrorIs(t, err, errCollateralCoinEmpty)
	_, err = e.GetEstimatedInterestAndBorrowable(t.Context(), testCrypto, testFiat, "", 0)
	assert.ErrorIs(t, err, errTermEmpty)
	_, err = e.GetEstimatedInterestAndBorrowable(t.Context(), testCrypto, testFiat, neigh, 0)
	assert.ErrorIs(t, err, errCollateralAmountEmpty)
	_, err = e.GetEstimatedInterestAndBorrowable(t.Context(), testCrypto, testFiat, "SEVEN", testPrice)
	assert.NoError(t, err)
}

func TestBorrowFunds(t *testing.T) {
	t.Parallel()
	_, err := e.BorrowFunds(t.Context(), currency.Code{}, currency.Code{}, "", 0, 0)
	assert.ErrorIs(t, err, errLoanCoinEmpty)
	_, err = e.BorrowFunds(t.Context(), testCrypto, currency.Code{}, "", 0, 0)
	assert.ErrorIs(t, err, errCollateralCoinEmpty)
	_, err = e.BorrowFunds(t.Context(), testCrypto, testFiat, "", 0, 0)
	assert.ErrorIs(t, err, errTermEmpty)
	_, err = e.BorrowFunds(t.Context(), testCrypto, testFiat, neigh, 0, 0)
	assert.ErrorIs(t, err, errCollateralLoanMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.BorrowFunds(t.Context(), testCrypto, testFiat, "SEVEN", testAmount, 0)
	assert.NoError(t, err)
}

func TestGetOngoingLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	resp, err := e.GetOngoingLoans(t.Context(), 0, currency.Code{}, currency.Code{})
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.GetOngoingLoans(t.Context(), resp[0].OrderID, currency.Code{}, currency.Code{})
	assert.NoError(t, err)
}

func TestGetLoanRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoanRepayHistory(t.Context(), 0, 0, 0, currency.Code{}, currency.Code{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetLoanRepayHistory(t.Context(), 0, 1, 5, currency.Code{}, currency.Code{}, time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestModifyPledgeRate(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyPledgeRate(t.Context(), 0, 0, currency.Code{}, "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.ModifyPledgeRate(t.Context(), 1, 0, currency.Code{}, "")
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.ModifyPledgeRate(t.Context(), 1, 1, currency.Code{}, "")
	assert.ErrorIs(t, err, errCollateralCoinEmpty)
	_, err = e.ModifyPledgeRate(t.Context(), 1, 1, currency.NewCode(meow), "")
	assert.ErrorIs(t, err, errReviseTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetOngoingLoans(t.Context(), 0, currency.Code{}, currency.Code{})
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.ModifyPledgeRate(t.Context(), resp[0].OrderID, testAmount, testFiat, "IN")
	assert.NoError(t, err)
}

func TestGetPledgeRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPledgeRateHistory(t.Context(), 0, 0, 0, "", currency.Code{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetPledgeRateHistory(t.Context(), 0, 1, 5, "", currency.Code{}, time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestGetLoanHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoanHistory(t.Context(), 0, 0, 0, currency.Code{}, currency.Code{}, "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetLoanHistory(t.Context(), 0, 1, 5, currency.Code{}, currency.Code{}, "", time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestGetDebts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	// If there aren't any debts to return information on, this will return the error "The data fetched by {user ID} is empty"
	testGetNoArgs(t, e.GetDebts)
}

func TestGetLiquidationRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidationRecords(t.Context(), 0, 0, 0, currency.Code{}, currency.Code{}, "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetLiquidationRecords(t.Context(), 0, 1, 5, currency.Code{}, currency.Code{}, "", time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestGetLoanInfo(t *testing.T) {
	t.Parallel()
	t.Skip(skipInstitution)
	testGetOneArg(t, e.GetLoanInfo, "", "1", errProductIDEmpty, false, true, true)
}

func TestGetMarginCoinRatio(t *testing.T) {
	t.Parallel()
	t.Skip(skipInstitution)
	testGetOneArg(t, e.GetMarginCoinRatio, "", "1", errProductIDEmpty, false, true, true)
}

func TestGetSpotSymbols(t *testing.T) {
	t.Parallel()
	t.Skip(skipInstitution)
	testGetOneArg(t, e.GetSpotSymbols, "", "1", errProductIDEmpty, false, true, true)
}

func TestGetLoanToValue(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	tarID := riskUnitHelper(t)
	_, err := e.GetLoanToValue(t.Context(), tarID)
	assert.NoError(t, err)
}

func TestGetTransferableAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetTransferableAmount(t.Context(), "", currency.Code{})
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetTransferableAmount(t.Context(), "", testFiat)
	assert.NoError(t, err)
}

func TestGetRiskUnit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetRiskUnit)
}

func TestSubaccountRiskUnitBinding(t *testing.T) {
	t.Parallel()
	_, err := e.SubaccountRiskUnitBinding(t.Context(), "", "", false)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	tarID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	tarID2 := riskUnitHelper(t)
	_, err = e.SubaccountRiskUnitBinding(t.Context(), tarID, tarID2, false)
	assert.NoError(t, err)
}

func TestGetLoanOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetLoanOrders(t.Context(), "", time.Now().Add(time.Minute), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetLoanOrders(t.Context(), "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetRepaymentOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetRepaymentOrders(t.Context(), 0, time.Now().Add(time.Minute), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetRepaymentOrders(t.Context(), 10, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.FetchTradablePairs, asset.Empty, asset.Spot, asset.ErrNotSupported, false, false, false)
	testGetOneArg(t, e.FetchTradablePairs, 0, asset.Futures, nil, false, false, false)
	testGetOneArg(t, e.FetchTradablePairs, 0, asset.Margin, nil, false, false, false)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.Pair{}, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.UpdateTicker(t.Context(), fakePair, asset.Spot)
	assert.Error(t, err)
	_, err = e.UpdateTicker(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = e.UpdateTicker(t.Context(), fakePair, asset.Futures)
	assert.Error(t, err)
	_, err = e.UpdateTicker(t.Context(), testPair, asset.Futures)
	assert.NoError(t, err)
	_, err = e.UpdateTicker(t.Context(), fakePair, asset.Margin)
	assert.Error(t, err)
	_, err = e.UpdateTicker(t.Context(), testPair, asset.Margin)
	assert.NoError(t, err)
	_, err = e.UpdateTicker(t.Context(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, e)
	err := e.UpdateTickers(t.Context(), asset.Spot)
	assert.NoError(t, err)
	err = e.UpdateTickers(t.Context(), asset.Futures)
	assert.NoError(t, err)
	err = e.UpdateTickers(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = e.UpdateTickers(t.Context(), asset.Margin)
	assert.NoError(t, err)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), currency.Pair{}, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.UpdateOrderbook(t.Context(), fakePair, asset.Spot)
	assert.Error(t, err)
	_, err = e.UpdateOrderbook(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = e.UpdateOrderbook(t.Context(), fakePair, asset.Futures)
	assert.Error(t, err)
	_, err = e.UpdateOrderbook(t.Context(), testPair, asset.Futures)
	assert.NoError(t, err)
	_, err = e.UpdateOrderbook(t.Context(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err := e.UpdateAccountInfo(t.Context(), asset.Spot)
	assert.NoError(t, err)
	_, err = e.UpdateAccountInfo(t.Context(), asset.Futures)
	assert.NoError(t, err)
	_, err = e.UpdateAccountInfo(t.Context(), asset.Margin)
	assert.NoError(t, err)
	_, err = e.UpdateAccountInfo(t.Context(), asset.CrossMargin)
	assert.NoError(t, err)
	_, err = e.UpdateAccountInfo(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	var fakeBitget Exchange
	_, err := fakeBitget.FetchAccountInfo(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.FetchAccountInfo(t.Context(), asset.Futures)
	assert.NoError(t, err)
	// When called by itself, the first call will update the account info, while the second call will return it from GetHoldings; we want coverage for both code paths
	_, err = e.FetchAccountInfo(t.Context(), asset.Futures)
	assert.NoError(t, err)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	testGetNoArgs(t, e.GetAccountFundingHistory)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err := e.GetWithdrawalsHistory(t.Context(), testCrypto, 0)
	assert.NoError(t, err)
	_, err = e.GetWithdrawalsHistory(t.Context(), fakeCurrency, 0)
	assert.Error(t, err)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), currency.Pair{}, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetRecentTrades(t.Context(), fakePair, asset.Spot)
	assert.Error(t, err)
	_, err = e.GetRecentTrades(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), fakePair, asset.Futures)
	assert.Error(t, err)
	_, err = e.GetRecentTrades(t.Context(), testPair, asset.Futures)
	assert.NoError(t, err)
	_, err = e.GetRecentTrades(t.Context(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), currency.Pair{}, asset.Spot, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetHistoricTrades(t.Context(), fakePair, asset.Spot, time.Time{}, time.Time{})
	assert.Error(t, err)
	_, err = e.GetHistoricTrades(t.Context(), testPair, asset.Spot, time.Now().Add(-time.Hour*24*7), time.Now())
	assert.NoError(t, err)
	_, err = e.GetHistoricTrades(t.Context(), fakePair, asset.Futures, time.Time{}, time.Time{})
	assert.Error(t, err)
	_, err = e.GetHistoricTrades(t.Context(), testPair, asset.Futures, time.Now().Add(-time.Hour*24*7), time.Now())
	assert.NoError(t, err)
	_, err = e.GetHistoricTrades(t.Context(), testPair, asset.Empty, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetServerTime, 0, 0, nil, false, false, true)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	var ord *order.Submit
	_, err := e.SubmitOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrSubmissionIsNil)
	ord = &order.Submit{
		Exchange:    e.Name,
		Pair:        testPair,
		AssetType:   asset.Binary,
		Side:        order.Sell,
		Type:        order.Limit,
		Amount:      testAmount,
		Price:       testPrice,
		TimeInForce: order.ImmediateOrCancel,
	}
	_, err = e.SubmitOrder(t.Context(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ord.AssetType = asset.Futures
	_, err = e.SubmitOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ord.AssetType = asset.Spot
	_, err = e.SubmitOrder(t.Context(), ord)
	assert.NoError(t, err)
	ord.AssetType = asset.CrossMargin
	ord.TimeInForce = order.UnknownTIF
	ord.Side = order.Buy
	ord.Amount = testAmount2
	ord.Price = testPrice2
	_, err = e.SubmitOrder(t.Context(), ord)
	assert.NoError(t, err)
	ord.AssetType = asset.Margin
	ord.AutoBorrow = true
	_, err = e.SubmitOrder(t.Context(), ord)
	assert.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "", testPair, asset.Empty)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	_, err = e.GetOrderInfo(t.Context(), "0", testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetOrderInfo(t.Context(), "1", testPair2, asset.Futures)
	assert.NoError(t, err)
	_, err = e.GetOrderInfo(t.Context(), "2", testPair, asset.Margin)
	assert.NoError(t, err)
	_, err = e.GetOrderInfo(t.Context(), "3", testPair, asset.CrossMargin)
	assert.NoError(t, err)
	resp, err := e.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) != 0 {
		_, err = e.GetOrderInfo(t.Context(), strconv.FormatUint(uint64(resp[0].OrderID), 10), testPair, asset.Spot)
		assert.NoError(t, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.NewCode(""), "", "")
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetDepositAddress(t.Context(), testCrypto, "", "")
	assert.NoError(t, err)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	var req *withdraw.Request
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), req)
	assert.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	req = &withdraw.Request{
		Crypto: withdraw.CryptoRequest{
			Address: testAddress,
			Chain:   testCrypto.String(),
		},
		Currency: testCrypto,
		Amount:   testAmount,
		Exchange: e.Name,
	}
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), req)
	assert.NoError(t, err)
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.WithdrawFiatFunds, nil, nil, common.ErrFunctionNotSupported, false, true, false)
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var req *order.MultiOrderRequest
	_, err := e.GetActiveOrders(t.Context(), req)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil)
	req = &order.MultiOrderRequest{
		AssetType: asset.Binary,
		Side:      order.Sell,
		Type:      order.Limit,
		Pairs:     []currency.Pair{testPair},
	}
	_, err = e.GetActiveOrders(t.Context(), req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	req.AssetType = asset.CrossMargin
	_, err = e.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Margin
	_, err = e.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Futures
	_, err = e.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{}
	_, err = e.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Spot
	_, err = e.GetActiveOrders(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	req.Pairs = []currency.Pair{}
	_, err = e.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{testPair}
	_, err = e.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var req *order.MultiOrderRequest
	_, err := e.GetOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil)
	req = &order.MultiOrderRequest{
		AssetType: asset.Binary,
		Side:      order.Sell,
		Type:      order.Limit,
		Pairs:     []currency.Pair{testPair},
	}
	_, err = e.GetOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	req.AssetType = asset.CrossMargin
	_, err = e.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Margin
	_, err = e.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Futures
	_, err = e.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{}
	_, err = e.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Spot
	_, err = e.GetOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	req.Pairs = []currency.Pair{}
	_, err = e.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{testPair}
	_, err = e.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	var fb *exchange.FeeBuilder
	_, err := e.GetFeeByType(t.Context(), fb)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	fb = &exchange.FeeBuilder{}
	_, err = e.GetFeeByType(t.Context(), fb)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	fb.Pair = testPair
	_, err = e.GetFeeByType(t.Context(), fb)
	assert.NoError(t, err)
	fb.IsMaker = true
	_, err = e.GetFeeByType(t.Context(), fb)
	assert.NoError(t, err)
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	err := e.ValidateAPICredentials(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandles(t.Context(), currency.Pair{}, asset.Spot, kline.Raw, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetHistoricCandles(t.Context(), testPair, asset.Spot, kline.OneHour, time.Now().Add(-time.Hour*24), time.Now())
	assert.NoError(t, err)
	_, err = e.GetHistoricCandles(t.Context(), testPair, asset.Futures, kline.OneHour, time.Now().Add(-time.Hour*24), time.Now())
	assert.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricCandlesExtended(t.Context(), currency.Pair{}, asset.Spot, kline.Raw, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetHistoricCandlesExtended(t.Context(), testPair, asset.Spot, kline.OneHour, time.Now().Add(-time.Hour*24), time.Now())
	assert.NoError(t, err)
	_, err = e.GetHistoricCandlesExtended(t.Context(), testPair, asset.Futures, kline.OneHour, time.Now().Add(-time.Hour*24), time.Now())
	assert.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetFuturesContractDetails, asset.Empty, asset.Empty, nil, false, false, true)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	var nilReq *fundingrate.LatestRateRequest
	_, err := e.GetLatestFundingRates(t.Context(), nilReq)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	req1, req2 := new(fundingrate.LatestRateRequest), new(fundingrate.LatestRateRequest)
	req2.Pair = testPair
	testGetOneArg(t, e.GetLatestFundingRates, req1, req2, currency.ErrCurrencyPairEmpty, false, false, true)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := e.UpdateOrderExecutionLimits(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	assert.NoError(t, err)
	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	assert.NoError(t, err)
	err = e.UpdateOrderExecutionLimits(t.Context(), asset.Margin)
	assert.NoError(t, err)
}

func TestUpdateCurrencyStates(t *testing.T) {
	t.Parallel()
	err := e.UpdateCurrencyStates(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetAvailableTransferChains, currency.EMPTYCODE, testCrypto, currency.ErrCurrencyCodeEmpty, false, false, true)
	_, err := e.GetAvailableTransferChains(t.Context(), fakeCurrency)
	assert.Error(t, err)
}

func TestGetMarginRatesHistory(t *testing.T) {
	t.Parallel()
	req1 := &margin.RateHistoryRequest{
		Asset: asset.Margin,
		Pair:  testPair,
	}
	req2 := new(margin.RateHistoryRequest)
	*req2 = *req1
	req2.StartDate = time.Now().Add(-time.Hour * 24 * 90)
	testGetOneArg(t, e.GetMarginRatesHistory, nil, req2, common.ErrNilPointer, false, true, true)
	req2.Asset = asset.CrossMargin
	testGetOneArg(t, e.GetMarginRatesHistory, req1, req2, common.ErrDateUnset, false, true, true)
	req1.Asset = asset.CrossMargin
	testGetOneArg(t, e.GetMarginRatesHistory, req1, nil, common.ErrDateUnset, false, true, false)
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionSummary(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Pair: testPair})
	assert.NoError(t, err)
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositions(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	req := &futures.PositionsRequest{
		Pairs: currency.Pairs{testPair, currency.NewPair(currency.BTC, currency.ETH)},
	}
	_, err = e.GetFuturesPositions(t.Context(), req)
	assert.NoError(t, err)
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	req := new(futures.PositionsRequest)
	testGetOneArg(t, e.GetFuturesPositionOrders, nil, req, common.ErrNilPointer, false, true, true)
	req.Pairs = currency.Pairs{testPair}
	testGetOneArg(t, e.GetFuturesPositionOrders, nil, req, nil, false, true, true)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	req := &fundingrate.HistoricalRatesRequest{
		Pair: testPair,
	}
	testGetOneArg(t, e.GetHistoricalFundingRates, nil, req, common.ErrNilPointer, false, false, false)
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	err := e.SetCollateralMode(t.Context(), 0, 0)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.GetCollateralMode, 0, 0, common.ErrFunctionNotSupported, false, true, false)
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	err := e.SetMarginType(t.Context(), 0, currency.Pair{}, 0)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = e.SetMarginType(t.Context(), asset.Futures, currency.Pair{}, margin.Isolated)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetMarginType(t.Context(), asset.Futures, testPair, margin.Multi)
	assert.NoError(t, err)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, e.ChangePositionMargin, nil, nil, common.ErrFunctionNotSupported, false, true, false)
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	err := e.SetLeverage(t.Context(), 0, currency.Pair{}, 0, 0, 0)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = e.SetLeverage(t.Context(), asset.Futures, currency.Pair{}, 0, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	err = e.SetLeverage(t.Context(), asset.Futures, testPair, 0, 1, order.Long)
	assert.NoError(t, err)
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), 0, currency.Pair{}, 0, 0)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = e.GetLeverage(t.Context(), asset.Futures, currency.Pair{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	_, err = e.GetLeverage(t.Context(), asset.Futures, testPair, 0, 0)
	assert.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = e.GetLeverage(t.Context(), asset.Futures, testPair, margin.Isolated, 0)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = e.GetLeverage(t.Context(), asset.Futures, testPair, margin.Isolated, order.Long)
	assert.NoError(t, err)
	_, err = e.GetLeverage(t.Context(), asset.Futures, testPair, margin.Isolated, order.Short)
	assert.NoError(t, err)
	_, err = e.GetLeverage(t.Context(), asset.Futures, testPair, margin.Multi, 0)
	assert.NoError(t, err)
	_, err = e.GetLeverage(t.Context(), asset.Margin, currency.Pair{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.GetLeverage(t.Context(), asset.Margin, testPair, 0, 0)
	assert.NoError(t, err)
	_, err = e.GetLeverage(t.Context(), asset.CrossMargin, currency.Pair{}, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.GetLeverage(t.Context(), asset.CrossMargin, testPair, 0, 0)
	assert.NoError(t, err)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: testCrypto.Item, Quote: testFiat.Item, Asset: asset.Futures})
	assert.NoError(t, err)
}

func TestCalculateUpdateOrderbookChecksum(t *testing.T) {
	t.Parallel()
	ord := orderbook.Book{
		Asks: orderbook.Levels{
			{
				StrPrice:  "3",
				StrAmount: "1",
			},
		},
		Bids: orderbook.Levels{
			{
				StrPrice:  "4",
				StrAmount: "1",
			},
		},
	}
	resp := e.CalculateUpdateOrderbookChecksum(&ord)
	assert.Equal(t, uint32(892106381), resp)
	ord.Asks = make(orderbook.Levels, 26)
	data := "3141592653589793238462643383279502884197169399375105"
	for i := range ord.Asks {
		ord.Asks[i] = orderbook.Level{
			StrPrice:  string(data[i*2]),
			StrAmount: string(data[i*2+1]),
		}
	}
	resp = e.CalculateUpdateOrderbookChecksum(&ord)
	assert.Equal(t, uint32(2945115267), resp)
}

// The following 20 tests aren't parallel due to collisions with each other, and some other tests
func TestCommitConversion(t *testing.T) {
	_, err := e.CommitConversion(t.Context(), currency.Code{}, currency.Code{}, "", 0, 0, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty)
	_, err = e.CommitConversion(t.Context(), testCrypto, testFiat, "", 0, 0, 0)
	assert.ErrorIs(t, err, errTraceIDEmpty)
	_, err = e.CommitConversion(t.Context(), testCrypto, testFiat, "1", 0, 0, 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	_, err = e.CommitConversion(t.Context(), testCrypto, testFiat, "1", 1, 1, 0)
	assert.ErrorIs(t, err, limits.ErrPriceBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetQuotedPrice(t.Context(), testCrypto, testFiat, testAmount, 0)
	require.NoError(t, err)
	_, err = e.CommitConversion(t.Context(), testCrypto, testFiat, resp.TraceID, resp.FromCoinSize.Float64(), resp.ToCoinSize.Float64(), resp.ConvertPrice.Float64())
	assert.NoError(t, err)
}

func TestModifyPlanSpotOrder(t *testing.T) {
	_, err := e.ModifyPlanSpotOrder(t.Context(), 0, "", "", 0, 0, 0)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.ModifyPlanSpotOrder(t.Context(), 0, meow, "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = e.ModifyPlanSpotOrder(t.Context(), 0, meow, woof, 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = e.ModifyPlanSpotOrder(t.Context(), 0, meow, "limit", 1, 0, 0)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = e.ModifyPlanSpotOrder(t.Context(), 0, meow, woof, 1, 0, 0)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ordID, err := e.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	if len(ordID.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := e.ModifyPlanSpotOrder(t.Context(), ordID.OrderList[0].OrderID, ordID.OrderList[0].ClientOrderID, "limit", testPrice, testPrice, testAmount)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestCancelPlanSpotOrder(t *testing.T) {
	_, err := e.CancelPlanSpotOrder(t.Context(), 0, "")
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ordID, err := e.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	require.NotNil(t, ordID)
	if len(ordID.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := e.CancelPlanSpotOrder(t.Context(), ordID.OrderList[0].OrderID, ordID.OrderList[0].ClientOrderID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyOrder(t *testing.T) {
	var ord *order.Modify
	_, err := e.ModifyOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrModifyOrderIsNil)
	ord = &order.Modify{
		Pair:      testPair,
		AssetType: 1<<31 - 1,
		OrderID:   meow,
	}
	_, err = e.ModifyOrder(t.Context(), ord)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	ord.OrderID = "0"
	_, err = e.ModifyOrder(t.Context(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ord.AssetType = asset.Futures
	_, err = e.ModifyOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ordID, err := e.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	if len(ordID.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	ord.OrderID = strconv.FormatUint(ordID.OrderList[0].OrderID, 10)
	ord.ClientOrderID = ordID.OrderList[0].ClientOrderID
	ord.Type = order.Limit
	ord.Price = testPrice
	ord.TriggerPrice = testPrice
	ord.Amount = testAmount
	ord.AssetType = asset.Spot
	_, err = e.ModifyOrder(t.Context(), ord)
	assert.NoError(t, err)
}

func TestCancelTriggerFuturesOrders(t *testing.T) {
	var ordList []OrderIDStruct
	_, err := e.CancelTriggerFuturesOrders(t.Context(), ordList, currency.Pair{}, "", "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ordList = append(ordList, OrderIDStruct{
		OrderID:       1,
		ClientOrderID: "a",
	})
	resp, err := e.CancelTriggerFuturesOrders(t.Context(), ordList, testPair2, testFiat2.String()+"-FUTURES", "", currency.Code{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestRepayLoan(t *testing.T) {
	_, err := e.RepayLoan(t.Context(), 0, 0, false, false)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	_, err = e.RepayLoan(t.Context(), 1, 0, false, false)
	assert.ErrorIs(t, err, limits.ErrAmountBelowMin)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetOngoingLoans(t.Context(), 0, currency.Code{}, currency.Code{})
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = e.RepayLoan(t.Context(), resp[0].OrderID, testAmount, false, false)
	assert.NoError(t, err)
	_, err = e.RepayLoan(t.Context(), resp[0].OrderID, 0, true, true)
	assert.NoError(t, err)
}

func TestModifyFuturesOrder(t *testing.T) {
	p := &ModifyFuturesOrderParams{}
	_, err := e.ModifyFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	p.OrderID = 1
	_, err = e.ModifyFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	p.Pair = testPair2
	_, err = e.ModifyFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	p.ProductType = meow
	_, err = e.ModifyFuturesOrder(t.Context(), p)
	assert.ErrorIs(t, err, errNewClientOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.ModifyFuturesOrder(t.Context(), &ModifyFuturesOrderParams{OrderID: 1, ClientOrderID: "a", ProductType: testFiat2.String() + "-FUTURES", NewClientOrderID: "a", Pair: testPair2, NewAmount: testAmount2 + 1, NewPrice: testPrice2 + 2, NewTakeProfit: testPrice2 + 1, NewStopLoss: testPrice2 / 10})
	assert.NoError(t, err)
}

func TestCancelFuturesOrder(t *testing.T) {
	_, err := e.CancelFuturesOrder(t.Context(), currency.Pair{}, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelFuturesOrder(t.Context(), testPair2, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = e.CancelFuturesOrder(t.Context(), testPair2, woof, "", currency.Code{}, 0)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelFuturesOrder(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "", testFiat2, 1)
	assert.NoError(t, err)
}

func TestBatchCancelFuturesOrders(t *testing.T) {
	_, err := e.BatchCancelFuturesOrders(t.Context(), nil, currency.Pair{}, "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orders := []OrderIDStruct{
		{
			OrderID:       1,
			ClientOrderID: "a",
		},
	}
	_, err = e.BatchCancelFuturesOrders(t.Context(), orders, testPair2, testFiat2.String()+"-FUTURES", testFiat2)
	assert.NoError(t, err)
}

func TestFlashClosePosition(t *testing.T) {
	_, err := e.FlashClosePosition(t.Context(), currency.Pair{}, "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.FlashClosePosition(t.Context(), testPair2, "", testFiat2.String()+"-FUTURES")
	assert.NoError(t, err)
}

func TestCancelAllFuturesOrders(t *testing.T) {
	_, err := e.CancelAllFuturesOrders(t.Context(), currency.Pair{}, "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelAllFuturesOrders(t.Context(), currency.Pair{}, testFiat2.String()+"-FUTURES", testFiat2, time.Second*60)
	assert.NoError(t, err)
}

func TestCancelOrder(t *testing.T) {
	var ord *order.Cancel
	err := e.CancelOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil)
	ord = &order.Cancel{
		OrderID:   meow,
		AssetType: 1<<31 - 1,
	}
	err = e.CancelOrder(t.Context(), ord)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	ord.OrderID = "0"
	err = e.CancelOrder(t.Context(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ord.AssetType = asset.Margin
	err = e.CancelOrder(t.Context(), ord)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	resp, err := e.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) != 0 {
		ord.OrderID = strconv.FormatUint(uint64(resp[0].OrderID), 10)
		ord.Pair = testPair
		ord.AssetType = asset.Spot
		ord.ClientOrderID = resp[0].ClientOrderID
		err = e.CancelOrder(t.Context(), ord)
		assert.NoError(t, err)
	}
	ord.OrderID = "1"
	ord.Pair = testPair2
	ord.AssetType = asset.Futures
	ord.ClientOrderID = "a"
	err = e.CancelOrder(t.Context(), ord)
	assert.NoError(t, err)
	ord.OrderID = "2"
	ord.Pair = testPair
	ord.AssetType = asset.CrossMargin
	ord.ClientOrderID = "b"
	err = e.CancelOrder(t.Context(), ord)
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	var ord *order.Cancel
	_, err := e.CancelAllOrders(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil)
	ord = &order.Cancel{
		AssetType: asset.Empty,
	}
	_, err = e.CancelAllOrders(t.Context(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	ord.AssetType = asset.Spot
	ord.Pair = testPair
	_, err = e.CancelAllOrders(t.Context(), ord)
	assert.NoError(t, err)
	ord.AssetType = asset.Futures
	ord.Pair = testPair2
	_, err = e.CancelAllOrders(t.Context(), ord)
	assert.NoError(t, err)
}

func TestCancelCrossOrder(t *testing.T) {
	_, err := e.CancelCrossOrder(t.Context(), currency.Pair{}, "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelCrossOrder(t.Context(), testPair, "", 0)
	assert.ErrorIs(t, err, errOrderIDMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelCrossOrder(t.Context(), testPair, "", 1)
	assert.NoError(t, err)
	_, err = e.CancelCrossOrder(t.Context(), testPair, "a", 0)
	assert.NoError(t, err)
}

func TestBatchCancelCrossOrders(t *testing.T) {
	_, err := e.BatchCancelCrossOrders(t.Context(), currency.Pair{}, nil)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.BatchCancelCrossOrders(t.Context(), testPair, nil)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.BatchCancelCrossOrders(t.Context(), testPair, []OrderIDStruct{{
		OrderID:       1,
		ClientOrderID: "a",
	}})
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	var orders []order.Cancel
	orders = append(orders, order.Cancel{
		AssetType: asset.Empty,
	})
	_, err := e.CancelBatchOrders(t.Context(), orders)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	orders[0].OrderID = "0"
	_, err = e.CancelBatchOrders(t.Context(), orders)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	orders = nil
	orders = append(orders, order.Cancel{
		AssetType:     asset.Spot,
		OrderID:       "1",
		ClientOrderID: "a",
		Pair:          testPair,
	}, order.Cancel{
		AssetType:     asset.Futures,
		OrderID:       "2",
		ClientOrderID: "b",
		Pair:          testPair2,
	}, order.Cancel{
		AssetType:     asset.Margin,
		OrderID:       "3",
		ClientOrderID: "c",
		Pair:          testPair,
	}, order.Cancel{
		AssetType:     asset.CrossMargin,
		OrderID:       "4",
		ClientOrderID: "d",
		Pair:          testPair,
	})
	_, err = e.CancelBatchOrders(t.Context(), orders)
	assert.NoError(t, err)
}

func TestCancelIsolatedOrder(t *testing.T) {
	_, err := e.CancelIsolatedOrder(t.Context(), currency.Pair{}, "", 0)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.CancelIsolatedOrder(t.Context(), testPair2, "", 0)
	assert.ErrorIs(t, err, errOrderIDMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.CancelIsolatedOrder(t.Context(), testPair, "", 1)
	assert.NoError(t, err)
	_, err = e.CancelIsolatedOrder(t.Context(), testPair, "a", 0)
	assert.NoError(t, err)
}

func TestBatchCancelIsolatedOrders(t *testing.T) {
	_, err := e.BatchCancelIsolatedOrders(t.Context(), currency.Pair{}, nil)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = e.BatchCancelIsolatedOrders(t.Context(), testPair2, nil)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	_, err = e.BatchCancelIsolatedOrders(t.Context(), testPair, []OrderIDStruct{{
		OrderID:       1,
		ClientOrderID: "a",
	}})
	assert.NoError(t, err)
}

func TestWsAuth(t *testing.T) {
	e.Websocket.SetCanUseAuthenticatedEndpoints(false)
	err := e.WsAuth(t.Context(), nil)
	assert.ErrorIs(t, err, errAuthenticatedWebsocketDisabled)
	if e.Websocket.IsEnabled() && !e.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(e) {
		t.Skip(websocket.ErrWebsocketNotEnabled.Error())
	}
	e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	var dialer gws.Dialer
	go func() {
		timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
		select {
		case resp := <-e.Websocket.DataHandler:
			t.Errorf("%+v\n%T\n", resp, resp)
		case <-timer.C:
		}
		timer.Stop()
		for {
			<-e.Websocket.DataHandler
		}
	}()
	err = e.WsAuth(t.Context(), &dialer)
	require.NoError(t, err)
	time.Sleep(sharedtestvalues.WebsocketResponseDefaultTimeout)
}

type getNoArgsResp interface {
	*TimeResp | *P2PMerInfoResp | []ConvertCoinsResp | []BGBConvertCoinsResp | []VIPFeeRateResp | []ExchangeRateResp | []DiscountRateResp | []SupCurrencyResp | float64 | *SavingsBalance | *SharkFinBalance | *DebtsResp | []AssetOverviewResp | bool | *SymbolsResp | []SubaccountAssetsResp | []exchange.FundingHistory | []string
}

type getNoArgsAssertNotEmpty[G getNoArgsResp] func(context.Context) (G, error)

func testGetNoArgs[G getNoArgsResp](t *testing.T, f getNoArgsAssertNotEmpty[G]) {
	t.Helper()
	_, err := f(t.Context())
	assert.NoError(t, err)
}

type getOneArgResp interface {
	[]WhaleNetFlowResp | *FundFlowResp | []WhaleFundFlowResp | *CrVirSubResp | []GetAPIKeyResp | []FundingAssetsResp | []BotAccAssetsResp | []ConvertBGBResp | []CoinInfoResp | []SymbolInfoResp | []TickerResp | string | *SubOrderResp | *BatchOrderResp | bool | *InterestRateResp | []FutureTickerResp | []FutureAccDetails | []SubaccountFuturesResp | []CrossAssetResp | *MaxBorrowCross | *MaxTransferCross | []IntRateMaxBorrowCross | []TierConfigCross | *FlashRepayCross | []IsoAssetResp | []IntRateMaxBorrowIso | []TierConfigIso | *MaxBorrowIso | *MaxTransferIso | []FlashRepayIso | []EarnAssets | *LoanCurList | currency.Pairs | time.Time | []futures.Contract | []fundingrate.LatestRateResponse | []string | *margin.RateHistoryResponse | *fundingrate.HistoricalRates | []futures.PositionResponse | *withdraw.ExchangeResponse | collateral.Mode | *margin.PositionChangeResponse | *LoanInfo | *MarginCoinRatio | *SpotSymbols
}

type getOneArgParam interface {
	string | uint64 | []string | bool | asset.Item | *fundingrate.LatestRateRequest | currency.Code | *margin.RateHistoryRequest | *fundingrate.HistoricalRatesRequest | *futures.PositionsRequest | *withdraw.Request | *margin.PositionChangeRequest | currency.Pair | []currency.Code | currency.Pairs | OnOffBool
}

type getOneArgGen[R getOneArgResp, P getOneArgParam] func(context.Context, P) (R, error)

func testGetOneArg[R getOneArgResp, P getOneArgParam](t *testing.T, f getOneArgGen[R, P], callErrCheck, callNoEErr P, tarErr error, checkResp, checkCreds, canManipOrders bool) {
	t.Helper()
	if tarErr != nil {
		_, err := f(t.Context(), callErrCheck)
		assert.ErrorIs(t, err, tarErr)
	}
	if checkCreds {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipOrders)
	}
	resp, err := f(t.Context(), callNoEErr)
	require.NoError(t, err)
	if checkResp {
		assert.NotEmpty(t, resp)
	}
}

type getTwoArgsResp interface {
	[]FutureTickerResp | []FundingTimeResp | []FuturesPriceResp | []FundingCurrentResp | []ContractConfigResp
}

type getTwoArgsPairProduct[G getTwoArgsResp] func(context.Context, currency.Pair, string) (G, error)

func testGetTwoArgs[G getTwoArgsResp](t *testing.T, f getTwoArgsPairProduct[G]) {
	t.Helper()
	_, err := f(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = f(t.Context(), currency.NewPairWithDelimiter(meow, woof, ""), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = f(t.Context(), testPair2, testFiat2.String()+"-FUTURES")
	assert.NoError(t, err)
}

func subAccTestHelper(t *testing.T, compString, ignoreString string) string {
	t.Helper()
	resp, err := e.GetVirtualSubaccounts(t.Context(), 25, 0, "")
	assert.NoError(t, err)
	require.NotEmpty(t, resp)
	tarID := ""
	for i := range resp.SubaccountList {
		if resp.SubaccountList[i].SubaccountName == compString &&
			resp.SubaccountList[i].SubaccountName != ignoreString {
			tarID = resp.SubaccountList[i].SubaccountUID
			break
		}
		if compString == "" && resp.SubaccountList[i].SubaccountName != ignoreString {
			tarID = resp.SubaccountList[i].SubaccountUID
			break
		}
	}
	if tarID == "" {
		t.Skipf(skipTestSubAccNotFound, compString, ignoreString)
	}
	return tarID
}

func exchangeBaseHelper(e *Exchange) error {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		return err
	}
	exchCfg, err := cfg.GetExchangeConfig("Bitget")
	if err != nil {
		return err
	}
	if apiKey != "" {
		exchCfg.API.Credentials.Key = apiKey
		exchCfg.API.Credentials.Secret = apiSecret
		exchCfg.API.Credentials.ClientID = clientID
		exchCfg.API.AuthenticatedSupport = true
		exchCfg.API.AuthenticatedWebsocketSupport = true
	}
	e.Websocket = sharedtestvalues.NewTestWebsocket()
	err = e.Setup(exchCfg)
	if err != nil {
		return err
	}
	return nil
}

func riskUnitHelper(t *testing.T) string {
	t.Helper()
	resp, err := e.GetRiskUnit(t.Context())
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientRiskUnits)
	}
	return resp[0]
}
