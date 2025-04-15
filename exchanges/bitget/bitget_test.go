package bitget

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stream/buffer"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
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
	ordersNotFound                     = "Orders not found"
	skipTestSubAccNotFound             = "appropriate sub-account (equals %v, not equals %v) not found, skipping"
	skipInsufficientAPIKeysFound       = "insufficient API keys found, skipping"
	skipInsufficientBalance            = "insufficient balance to place order, skipping"
	skipInsufficientOrders             = "insufficient orders found, skipping"
	skipInsufficientRiskUnits          = "insufficient risk units found, skipping"
	skipInstitution                    = "this endpoint requires IDs tailored to an institution, so it can't be automatically tested, skipping"
	errAPIKeyLimitPartial              = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"40063","msg":"API exceeds the maximum limit added","requestTime":`
	errCurrentlyHoldingPositionPartial = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"45117","msg":"Currently holding positions or orders, the margin mode cannot be adjusted","requestTime":`
)

// Developer-defined variables to aid testing
var (
	fakeCurrency = currency.NewCode("FAKECURRENCYNOT")
	fakePair     = currency.NewPair(fakeCurrency, currency.NewCode("REALMEOWMEOW"))

	errUnmarshalArray string
)

var bi = &Bitget{}

func TestMain(m *testing.M) {
	bi.SetDefaults()
	err := exchangeBaseHelper(bi)
	if err != nil {
		log.Fatal(err)
	}
	var dialer websocket.Dialer
	err = bi.Websocket.Conn.Dial(&dialer, http.Header{})
	if err != nil {
		log.Fatal(err)
	}
	bi.Websocket.Wg.Add(1)
	go bi.wsReadData(bi.Websocket.Conn)
	bi.Websocket.Conn.SetupPingHandler(request.Unset, stream.PingHandler{
		Websocket:   true,
		Message:     []byte(`ping`),
		MessageType: websocket.TextMessage,
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

func TestInterface(t *testing.T) {
	t.Parallel()
	var e exchange.IBotExchange
	e = new(Bitget)
	_, ok := e.(exchange.IBotExchange)
	assert.True(t, ok)
}

func TestSetup(t *testing.T) {
	cfg, err := bi.GetStandardConfig()
	assert.NoError(t, err)
	exch := &Bitget{}
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
	assert.ErrorIs(t, err, stream.ErrWebsocketAlreadyInitialised)
}

func TestWsConnect(t *testing.T) {
	exch := &Bitget{}
	exch.Websocket = sharedtestvalues.NewTestWebsocket()
	err := exch.Websocket.Disable()
	assert.ErrorIs(t, err, stream.ErrAlreadyDisabled)
	err = exch.WsConnect()
	assert.ErrorIs(t, err, stream.ErrWebsocketNotEnabled)
	exch.SetDefaults()
	err = exchangeBaseHelper(exch)
	require.NoError(t, err)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	exch.Verbose = true
	err = exch.WsConnect()
	assert.NoError(t, err)
}

func TestQueryAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := bi.QueryAnnouncements(t.Context(), "", time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := bi.QueryAnnouncements(t.Context(), "latest_news", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetTime(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetTime)
}

func TestGetTradeRate(t *testing.T) {
	t.Parallel()
	_, err := bi.GetTradeRate(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetTradeRate(t.Context(), testPair, "")
	assert.ErrorIs(t, err, errBusinessTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetTradeRate(t.Context(), testPair, "spot")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotTransactionRecords(t.Context(), currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotTransactionRecords(t.Context(), testFiat, time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetFuturesTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesTransactionRecords(t.Context(), "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesTransactionRecords(t.Context(), "woof", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesTransactionRecords(t.Context(), "COIN-FUTURES", currency.Code{}, time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetMarginTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMarginTransactionRecords(t.Context(), "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetMarginTransactionRecords(t.Context(), "crossed", testFiat, time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetP2PTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetP2PTransactionRecords(t.Context(), currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetP2PTransactionRecords(t.Context(), testFiat, time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetP2PMerchantList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetP2PMerchantList(t.Context(), "", 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetMerchantInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetMerchantInfo)
}

func TestGetMerchantP2POrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMerchantP2POrders(t.Context(), time.Time{}, time.Time{}, 0, 0, 0, 0, "", "", currency.Code{}, currency.Code{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// Can't currently be properly tested due to not knowing any p2p order IDs
	_, err = bi.GetMerchantP2POrders(t.Context(), time.Now().Add(-time.Hour*24*7), time.Now(), 5, 1, 0, 0, "", "", testCrypto, currency.Code{})
	assert.NoError(t, err)
}

func TestGetMerchantAdvertisementList(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMerchantAdvertisementList(t.Context(), time.Time{}, time.Time{}, 0, 0, 0, 0, "", "", "", "", currency.Code{}, currency.Code{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetMerchantAdvertisementList(t.Context(), time.Now().Add(-time.Hour*24*7), time.Now(), 5, 1<<62, 0, 0, "", "sell", "", "", testCrypto, currency.USD)
	assert.NoError(t, err)
}

func TestGetSpotWhaleNetFlow(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetSpotWhaleNetFlow, currency.Pair{}, testPair, errPairEmpty, true, false, false)
}

func TestGetFuturesActiveVolume(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesActiveVolume(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetFuturesActiveVolume(t.Context(), testPair, "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesPositionRatios(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesPositionRatios(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetFuturesPositionRatios(t.Context(), testPair, "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetMarginPositionRatios(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMarginPositionRatios(t.Context(), currency.Pair{}, "", currency.Code{})
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetMarginPositionRatios(t.Context(), testPair, "24h", currency.Code{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetMarginLoanGrowth(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMarginLoanGrowth(t.Context(), currency.Pair{}, "", currency.Code{})
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetMarginLoanGrowth(t.Context(), testPair, "24h", currency.Code{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetIsolatedBorrowingRatio(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedBorrowingRatio(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetIsolatedBorrowingRatio(t.Context(), testPair, "24h")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesRatios(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesRatios(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetFuturesRatios(t.Context(), testPair, "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotFundFlows(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotFundFlows(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetSpotFundFlows(t.Context(), testPair, "15m")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetTradeSupportSymbols(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetTradeSupportSymbols)
}

func TestGetSpotWhaleFundFlows(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetSpotWhaleFundFlows, currency.Pair{}, testPair, errPairEmpty, true, false, false)
}

func TestGetFuturesAccountRatios(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesAccountRatios(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetFuturesAccountRatios(t.Context(), testPair, "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestCreateVirtualSubaccounts(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.CreateVirtualSubaccounts, nil, []string{testSubaccountName}, errSubaccountEmpty, true, true, true)
}

func TestModifyVirtualSubaccount(t *testing.T) {
	t.Parallel()
	perms := []string{}
	_, err := bi.ModifyVirtualSubaccount(t.Context(), "", "", perms)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.ModifyVirtualSubaccount(t.Context(), "meow", "", perms)
	assert.ErrorIs(t, err, errNewStatusEmpty)
	_, err = bi.ModifyVirtualSubaccount(t.Context(), "meow", "woof", perms)
	assert.ErrorIs(t, err, errNewPermsEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	tarID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	perms = append(perms, "read")
	resp, err := bi.ModifyVirtualSubaccount(t.Context(), tarID, "normal", perms)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestCreateSubaccountAndAPIKey(t *testing.T) {
	t.Parallel()
	ipL := []string{}
	_, err := bi.CreateSubaccountAndAPIKey(t.Context(), "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ipL = append(ipL, testIP)
	pL := []string{"read"}
	// Fails with error "subAccountList not empty" and I'm not sure why. The account I'm testing with is far off hitting the limit of 20 sub-accounts.
	// Now it's saying that parameter req cannot be empty, still no clue what that means
	// Now it's saying that parameter verification failed. Occasionally it says this in Chinese
	_, err = bi.CreateSubaccountAndAPIKey(t.Context(), "MEOWMEOW", "woofwoof123", "neighneighneighneighneigh", ipL, pL)
	assert.NoError(t, err)
}

func TestGetVirtualSubaccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetVirtualSubaccounts(t.Context(), 25, 1, "")
	assert.NoError(t, err)
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	ipL := []string{}
	_, err := bi.CreateAPIKey(t.Context(), "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.CreateAPIKey(t.Context(), "woof", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errPassphraseEmpty)
	_, err = bi.CreateAPIKey(t.Context(), "woof", "meow", "", ipL, ipL)
	assert.ErrorIs(t, err, errLabelEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	tarID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	ipL = append(ipL, testIP)
	pL := []string{"read"}
	_, err = bi.CreateAPIKey(t.Context(), tarID, clientID, "neigh whinny", ipL, pL)
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
	_, err := bi.ModifyAPIKey(t.Context(), "", "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errAPIKeyEmpty)
	_, err = bi.ModifyAPIKey(t.Context(), "", "", "", "woof", ipL, ipL)
	assert.ErrorIs(t, err, errPassphraseEmpty)
	_, err = bi.ModifyAPIKey(t.Context(), "", "meow", "", "woof", ipL, ipL)
	assert.ErrorIs(t, err, errLabelEmpty)
	_, err = bi.ModifyAPIKey(t.Context(), "", "meow", "quack", "woof", ipL, ipL)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	tarID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	resp, err := bi.GetAPIKeys(t.Context(), tarID)
	assert.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientAPIKeysFound)
	}
	resp2, err := bi.ModifyAPIKey(t.Context(), tarID, clientID, "oink", resp[0].SubaccountAPIKey, ipL, ipL)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2)
}

func TestGetAPIKeys(t *testing.T) {
	t.Parallel()
	var tarID string
	if sharedtestvalues.AreAPICredentialsSet(bi) {
		tarID = subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	}
	testGetOneArg(t, bi.GetAPIKeys, "", tarID, errSubaccountEmpty, true, true, true)
}

func TestGetFundingAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetFundingAssets, currency.Code{}, testCrypto, nil, false, true, true)
}

func TestGetBotAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetBotAccountAssets, "", "spot", nil, false, true, true)
}

func TestGetAssetOverview(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetAssetOverview)
}

func TestGetConvertCoints(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetConvertCoins)
}

func TestGetQuotedPrice(t *testing.T) {
	t.Parallel()
	_, err := bi.GetQuotedPrice(t.Context(), currency.Code{}, currency.Code{}, 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.GetQuotedPrice(t.Context(), currency.NewCode("meow"), currency.NewCode("woof"), 0, 0)
	assert.ErrorIs(t, err, errFromToMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetQuotedPrice(t.Context(), testCrypto, testFiat, 0, 1)
	assert.NoError(t, err)
	resp, err := bi.GetQuotedPrice(t.Context(), testCrypto, testFiat, 0.1, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetConvertHistory(t.Context(), time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetConvertHistory(t.Context(), time.Now().Add(-time.Hour*90*24), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetBGBConvertCoins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetBGBConvertCoins)
}

func TestConvertBGB(t *testing.T) {
	t.Parallel()
	// No matter what currency I use, this returns the error "currency does not support convert"; possibly a bad error message, with the true issue being lack of funds?
	testGetOneArg(t, bi.ConvertBGB, nil, []currency.Code{testCrypto3}, errCurrencyEmpty, false, true, canManipulateRealOrders)
}

func TestGetBGBConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetBGBConvertHistory(t.Context(), 0, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetBGBConvertHistory(t.Context(), 0, 5, 1<<62, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetCoinInfo(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCoinInfo, currency.Code{}, testCrypto, nil, true, false, false)
}

func TestGetSymbolInfo(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetSymbolInfo, currency.Pair{}, currency.Pair{}, nil, true, false, false)
}

func TestGetSpotVIPFeeRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetSpotVIPFeeRate)
}

func TestGetSpotTickerInformation(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetSpotTickerInformation, currency.Pair{}, testPair, nil, true, false, false)
}

func TestGetSpotMergeDepth(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotMergeDepth(t.Context(), currency.Pair{}, "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetSpotMergeDepth(t.Context(), testPair, "scale3", "5")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetOrderbookDepth(t *testing.T) {
	t.Parallel()
	resp, err := bi.GetOrderbookDepth(t.Context(), testPair, "step0", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotCandlestickData(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotCandlestickData(t.Context(), currency.Pair{}, "", time.Time{}, time.Time{}, 0, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotCandlestickData(t.Context(), testPair, "", time.Time{}, time.Time{}, 0, false)
	assert.ErrorIs(t, err, errGranEmpty)
	_, err = bi.GetSpotCandlestickData(t.Context(), testPair, "woof", time.Time{}, time.Time{}, 5, true)
	assert.ErrorIs(t, err, errEndTimeEmpty)
	_, err = bi.GetSpotCandlestickData(t.Context(), testPair, "woof", time.Now().Add(time.Hour), time.Time{}, 0, false)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	_, err = bi.GetSpotCandlestickData(t.Context(), testPair, "1min", time.Time{}, time.Time{}, 5, false)
	assert.NoError(t, err)
	resp, err := bi.GetSpotCandlestickData(t.Context(), testPair, "1min", time.Time{}, time.Now(), 5, true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.SpotCandles)
}

func TestGetRecentSpotFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetRecentSpotFills(t.Context(), currency.Pair{}, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetRecentSpotFills(t.Context(), testPair, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotMarketTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotMarketTrades(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotMarketTrades(t.Context(), testPair, time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := bi.GetSpotMarketTrades(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestPlaceSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceSpotOrder(t.Context(), currency.Pair{}, "", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, false, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceSpotOrder(t.Context(), testPair, "", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, false, 0)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceSpotOrder(t.Context(), testPair, "sell", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, false, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceSpotOrder(t.Context(), testPair, "sell", "limit", "", "", "", 0, 0, 0, 0, 0, 0, 0, false, 0)
	assert.ErrorIs(t, err, errStrategyEmpty)
	_, err = bi.PlaceSpotOrder(t.Context(), testPair, "sell", "limit", "IOC", "", "", 0, 0, 0, 0, 0, 0, 0, false, 0)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = bi.PlaceSpotOrder(t.Context(), testPair, "sell", "limit", "IOC", "", "", testPrice, 0, 0, 0, 0, 0, 0, false, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceSpotOrder(t.Context(), testPair, "sell", "limit", "IOC", "", "", testPrice, testAmount, 0, testPrice-1, testPrice-2, testPrice+1, testPrice+2, true, 0)
	assert.NoError(t, err)
	_, err = bi.PlaceSpotOrder(t.Context(), testPair, "sell", "limit", "IOC", "", "", testPrice, testAmount, testPrice/10, 0, 0, 0, 0, false, time.Minute)
	assert.NoError(t, err)
}

func TestCancelAndPlaceSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.CancelAndPlaceSpotOrder(t.Context(), currency.Pair{}, "", "", 0, 0, 0, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelAndPlaceSpotOrder(t.Context(), testPair, "", "", 0, 0, 0, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.CancelAndPlaceSpotOrder(t.Context(), testPair, "meow", "", 0, 0, 0, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	cID := clientIDGenerator()
	_, err = bi.CancelAndPlaceSpotOrder(t.Context(), testPair, resp[0].ClientOrderID, cID, testPrice, testAmount, testPrice-1, testPrice-2, testPrice+1, testPrice+2, int64(resp[0].OrderID))
	assert.NoError(t, err)
}

func TestBatchCancelAndPlaceSpotOrders(t *testing.T) {
	t.Parallel()
	var req []ReplaceSpotOrderStruct
	_, err := bi.BatchCancelAndPlaceSpotOrders(t.Context(), req)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	newPair, err := pairFromStringHelper(resp[0].Symbol)
	require.NoError(t, err)
	req = append(req, ReplaceSpotOrderStruct{
		OrderID:          int64(resp[0].OrderID),
		OldClientOrderID: resp[0].ClientOrderID,
		Price:            testPrice,
		Amount:           testAmount,
		Pair:             newPair,
	})
	_, err = bi.BatchCancelAndPlaceSpotOrders(t.Context(), req)
	assert.NoError(t, err)
}

func TestCancelSpotOrderByID(t *testing.T) {
	t.Parallel()
	_, err := bi.CancelSpotOrderByID(t.Context(), currency.Pair{}, "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelSpotOrderByID(t.Context(), testPair, "", "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.CancelSpotOrderByID(t.Context(), testPair, "", resp[0].ClientOrderID, int64(resp[0].OrderID))
	assert.NoError(t, err)
}

func TestBatchPlaceSpotOrders(t *testing.T) {
	t.Parallel()
	var req []PlaceSpotOrderStruct
	_, err := bi.BatchPlaceSpotOrders(t.Context(), currency.Pair{}, false, false, req)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchPlaceSpotOrders(t.Context(), testPair, false, false, req)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	req = append(req, PlaceSpotOrderStruct{
		Side:      "sell",
		OrderType: "limit",
		Strategy:  "IOC",
		Price:     testPrice,
		Size:      testAmount,
		Pair:      testPair,
	})
	resp, err := bi.BatchPlaceSpotOrders(t.Context(), testPair, true, true, req)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestBatchCancelOrders(t *testing.T) {
	t.Parallel()
	var req []CancelSpotOrderStruct
	_, err := bi.BatchCancelOrders(t.Context(), currency.Pair{}, false, req)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchCancelOrders(t.Context(), testPair, false, req)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	pair, err := pairFromStringHelper(resp[0].Symbol)
	assert.NoError(t, err)
	req = append(req, CancelSpotOrderStruct{
		OrderID:       int64(resp[0].OrderID),
		ClientOrderID: resp[0].ClientOrderID,
		Pair:          pair,
	})
	resp2, err := bi.BatchCancelOrders(t.Context(), testPair, true, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2)
}

func TestCancelOrderBySymbol(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.CancelOrdersBySymbol, currency.Pair{}, testPair, errPairEmpty, true, true, canManipulateRealOrders)
}

func TestGetSpotOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotOrderDetails(t.Context(), 0, "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotOrderDetails(t.Context(), 1, "a", time.Minute)
	assert.NoError(t, err)
}

func TestGetUnfilledOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetUnfilledOrders(t.Context(), currency.Pair{}, "", time.Now().Add(time.Hour), time.Time{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetUnfilledOrders(t.Context(), currency.Pair{}, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	assert.NoError(t, err)
}

func TestGetHistoricalSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalSpotOrders(t.Context(), currency.Pair{}, time.Now().Add(time.Hour), time.Time{}, 0, 0, 0, "", time.Minute)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetHistoricalSpotOrders(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 5, 1<<62, 0, "", time.Minute)
	assert.NoError(t, err)
}

func TestGetSpotFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotFills(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotFills(t.Context(), testPair, time.Now().Add(time.Hour), time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotFills(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
}

func TestPlacePlanSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlacePlanSpotOrder(t.Context(), currency.Pair{}, "", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlacePlanSpotOrder(t.Context(), testPair, "", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlacePlanSpotOrder(t.Context(), testPair, "woof", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.PlacePlanSpotOrder(t.Context(), testPair, "woof", "", "", "", "", "", "", 1, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlacePlanSpotOrder(t.Context(), testPair, "woof", "limit", "", "", "", "", "", 1, 0, 0)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = bi.PlacePlanSpotOrder(t.Context(), testPair, "woof", "neigh", "", "", "", "", "", 1, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.PlacePlanSpotOrder(t.Context(), currency.NewPairWithDelimiter("meow", "woof", ""), "neigh", "oink", "", "", "", "", "", 1, 0, 1)
	assert.ErrorIs(t, err, errTriggerTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.PlacePlanSpotOrder(t.Context(), testPair, "sell", "limit", "", "fill_price", clientIDGenerator(), "ioc", "none", testPrice, testPrice, testAmount)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrentSpotPlanOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCurrentSpotPlanOrders(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestSpotGetPlanSubOrder(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// This gets the error "the current plan order does not exist or has not been triggered" even when using a plan order that definitely exists and has definitely been triggered. Re-investigate later
	testGetOneArg(t, bi.GetSpotPlanSubOrder, 0, 1, errOrderIDEmpty, true, true, true)
}

func TestGetSpotPlanOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotPlanOrderHistory(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotPlanOrderHistory(t.Context(), testPair, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotPlanOrderHistory(t.Context(), testPair, time.Now().Add(-time.Hour*24*90), time.Now().Add(-time.Minute), 2, 1<<62)
	assert.NoError(t, err)
}

func TestBatchCancelSpotPlanOrders(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.BatchCancelSpotPlanOrders, nil, currency.Pairs{testPair}, nil, false, true, canManipulateRealOrders)
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// Not chucked into testGetNoArgs due to checking the presence of resp.Data, refactoring that generic for that would waste too many lines to do so just for this
	resp, err := bi.GetAccountInfo(t.Context())
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetAccountAssets(t.Context(), testCrypto, "all")
	assert.NoError(t, err)
}

func TestGetSpotSubaccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetSpotSubaccountAssets)
}

func TestModifyDepositAccount(t *testing.T) {
	t.Parallel()
	_, err := bi.ModifyDepositAccount(t.Context(), "", currency.Code{})
	assert.ErrorIs(t, err, errAccountTypeEmpty)
	_, err = bi.ModifyDepositAccount(t.Context(), "meow", currency.Code{})
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ModifyDepositAccount(t.Context(), "spot", testFiat)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAccountBills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotAccountBills(t.Context(), currency.Code{}, "", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotAccountBills(t.Context(), testCrypto, "", "", time.Time{}, time.Time{}, 3, 1<<62)
	assert.NoError(t, err)
}

func TestTransferAsset(t *testing.T) {
	t.Parallel()
	_, err := bi.TransferAsset(t.Context(), "", "", "", currency.Code{}, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.TransferAsset(t.Context(), "meow", "", "", currency.Code{}, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errToTypeEmpty)
	_, err = bi.TransferAsset(t.Context(), "meow", "woof", "", currency.Code{}, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errCurrencyAndPairEmpty)
	_, err = bi.TransferAsset(t.Context(), "meow", "woof", "", currency.Code{}, testPair, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.TransferAsset(t.Context(), "spot", "p2p", clientIDGenerator(), testCrypto, testPair, testAmount)
	assert.NoError(t, err)
}

func TestGetTransferableCoinList(t *testing.T) {
	t.Parallel()
	_, err := bi.GetTransferableCoinList(t.Context(), "", "")
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.GetTransferableCoinList(t.Context(), "meow", "")
	assert.ErrorIs(t, err, errToTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetTransferableCoinList(t.Context(), "spot", "p2p")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestSubaccountTransfer(t *testing.T) {
	t.Parallel()
	_, err := bi.SubaccountTransfer(t.Context(), "", "", "", "", "", currency.Code{}, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.SubaccountTransfer(t.Context(), "meow", "", "", "", "", currency.Code{}, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errToTypeEmpty)
	_, err = bi.SubaccountTransfer(t.Context(), "meow", "woof", "", "", "", currency.Code{}, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errCurrencyAndPairEmpty)
	_, err = bi.SubaccountTransfer(t.Context(), "meow", "woof", "", "", "", testCrypto, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errFromIDEmpty)
	_, err = bi.SubaccountTransfer(t.Context(), "meow", "woof", "", "neigh", "", testCrypto, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errToIDEmpty)
	_, err = bi.SubaccountTransfer(t.Context(), "meow", "woof", "", "neigh", "moo", testCrypto, currency.Pair{}, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	fromID := subAccTestHelper(t, "", strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com")
	toID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	_, err = bi.SubaccountTransfer(t.Context(), "spot", "p2p", clientIDGenerator(), fromID, toID, testCrypto, testPair, testAmount)
	assert.NoError(t, err)
}

func TestWithdrawFunds(t *testing.T) {
	t.Parallel()
	_, err := bi.WithdrawFunds(t.Context(), currency.Code{}, "", "", "", "", "", "", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.WithdrawFunds(t.Context(), testCrypto, "", "", "", "", "", "", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errTransferTypeEmpty)
	_, err = bi.WithdrawFunds(t.Context(), testCrypto, "woof", "", "", "", "", "", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errAddressEmpty)
	_, err = bi.WithdrawFunds(t.Context(), testCrypto, "woof", "neigh", "", "", "", "", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.WithdrawFunds(t.Context(), testCrypto, "on_chain", testAddress, testCrypto.String(), "", "", "", "", clientIDGenerator(), "", "", "", "", "", testAmount)
	assert.NoError(t, err)
}

func TestGetSubaccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSubaccountTransferRecord(t.Context(), currency.Code{}, "", "", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSubaccountTransferRecord(t.Context(), testCrypto, "", "meow", "initiator", time.Time{}, time.Time{}, 3, 1<<62)
	assert.NoError(t, err)
}

func TestGetTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := bi.GetTransferRecord(t.Context(), currency.Code{}, "", "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.GetTransferRecord(t.Context(), testCrypto, "", "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.GetTransferRecord(t.Context(), testCrypto, "woof", "", time.Now().Add(time.Hour), time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetTransferRecord(t.Context(), testCrypto, "spot", "meow", time.Time{}, time.Time{}, 3, 1<<62, 1)
	assert.NoError(t, err)
}

func TestSwitchBGBDeductionStatus(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.SwitchBGBDeductionStatus, false, false, nil, false, true, canManipulateRealOrders)
	testGetOneArg(t, bi.SwitchBGBDeductionStatus, false, true, nil, false, true, canManipulateRealOrders)
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	_, err := bi.GetDepositAddressForCurrency(t.Context(), currency.Code{}, "", 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetDepositAddressForCurrency(t.Context(), testCrypto, "", 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSubaccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSubaccountDepositAddress(t.Context(), "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.GetSubaccountDepositAddress(t.Context(), "meow", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSubaccountDepositAddress(t.Context(), deposSubaccID, "", testCrypto, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetBGBDeductionStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetBGBDeductionStatus)
}

func TestCancelWithdrawal(t *testing.T) {
	t.Parallel()
	_, err := bi.CancelWithdrawal(t.Context(), 0)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.WithdrawFunds(t.Context(), testCrypto, "on_chain", testAddress, testCrypto.String(), "", "", "", "", clientIDGenerator(), "", "", "", "", "", testAmount)
	require.NoError(t, err)
	require.NotEmpty(t, resp.OrderID)
	_, err = bi.CancelWithdrawal(t.Context(), int64(resp.OrderID))
	assert.NoError(t, err)
}

func TestGetSubaccountDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSubaccountDepositRecords(t.Context(), "", currency.Code{}, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.GetSubaccountDepositRecords(t.Context(), "meow", currency.Code{}, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := subAccTestHelper(t, "", "")
	_, err = bi.GetSubaccountDepositRecords(t.Context(), tarID, currency.Code{}, 1<<62, 2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetWithdrawalRecords(t.Context(), currency.Code{}, "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetWithdrawalRecords(t.Context(), testCrypto, "", time.Now().Add(-time.Hour*24*90), time.Now(), 1<<62, 0, 5)
	assert.NoError(t, err)
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetDepositRecords(t.Context(), currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetDepositRecords(t.Context(), testCrypto, 0, 1<<62, 2, time.Now().Add(-time.Hour*24*90), time.Now())
	assert.NoError(t, err)
}

func TestGetFuturesVIPFeeRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetFuturesVIPFeeRate)
}

func TestGetInterestRateHistory(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetInterestRateHistory, currency.Code{}, testFiat, errCurrencyEmpty, true, false, false)
}

func TestGetInterestExchangeRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetInterestExchangeRate)
}

func TestGetDiscountRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetDiscountRate)
}

func TestGetFuturesMergeDepth(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesMergeDepth(t.Context(), currency.Pair{}, "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesMergeDepth(t.Context(), testPair, "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetFuturesMergeDepth(t.Context(), testPair, "USDT-FUTURES", "scale3", "5")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesTicker(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, bi.GetFuturesTicker)
}

func TestGetAllFuturesTickers(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetAllFuturesTickers, "", "COIN-FUTURES", errProductTypeEmpty, true, false, false)
}

func TestGetRecentFuturesFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetRecentFuturesFills(t.Context(), currency.Pair{}, "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetRecentFuturesFills(t.Context(), testPair, "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetRecentFuturesFills(t.Context(), testPair, "USDT-FUTURES", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesMarketTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesMarketTrades(t.Context(), currency.Pair{}, "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesMarketTrades(t.Context(), testPair, "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesMarketTrades(t.Context(), testPair, "woof", 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := bi.GetFuturesMarketTrades(t.Context(), testPair, "USDT-FUTURES", 5, 1<<62, time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesCandlestickData(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesCandlestickData(t.Context(), currency.Pair{}, "", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesCandlestickData(t.Context(), testPair, "", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesCandlestickData(t.Context(), testPair, "woof", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errGranEmpty)
	_, err = bi.GetFuturesCandlestickData(t.Context(), testPair, "woof", "neigh", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := bi.GetFuturesCandlestickData(t.Context(), testPair, "USDT-FUTURES", "1m", "", time.Time{}, time.Time{}, 5, CallModeNormal)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.FuturesCandles)
	resp, err = bi.GetFuturesCandlestickData(t.Context(), testPair, "COIN-FUTURES", "1m", "", time.Time{}, time.Time{}, 5, CallModeHistory)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.FuturesCandles)
	resp, err = bi.GetFuturesCandlestickData(t.Context(), testPair, "USDC-FUTURES", "1m", "", time.Time{}, time.Now(), 5, CallModeIndex)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.FuturesCandles)
	resp, err = bi.GetFuturesCandlestickData(t.Context(), testPair, "USDT-FUTURES", "1m", "", time.Time{}, time.Now(), 5, CallModeMark)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.FuturesCandles)
}

func TestGetOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := bi.GetOpenPositions(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = bi.GetOpenPositions(t.Context(), currency.NewPairWithDelimiter("meow", "woof", ""), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetOpenPositions(t.Context(), testPair2, testFiat2.String()+"-FUTURES")
	assert.NoError(t, err)
}

func TestGetNextFundingTime(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, bi.GetNextFundingTime)
}

func TestGetFuturesPrices(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, bi.GetFuturesPrices)
}

func TestGetFundingHistorical(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFundingHistorical(t.Context(), currency.Pair{}, "", 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFundingHistorical(t.Context(), testPair, "", 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetFundingHistorical(t.Context(), testPair, "USDT-FUTURES", 5, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFundingCurrent(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, bi.GetFundingCurrent)
}

func TestGetContractConfig(t *testing.T) {
	t.Parallel()
	_, err := bi.GetContractConfig(t.Context(), currency.Pair{}, "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetContractConfig(t.Context(), currency.Pair{}, prodTypes[0])
	assert.NoError(t, err)
}

func TestGetOneFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := bi.GetOneFuturesAccount(t.Context(), currency.Pair{}, "", currency.Code{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetOneFuturesAccount(t.Context(), testPair, "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetOneFuturesAccount(t.Context(), testPair, "woof", currency.Code{})
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetOneFuturesAccount(t.Context(), testPair, "USDT-FUTURES", testFiat)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAllFuturesAccounts(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetAllFuturesAccounts, "", "COIN-FUTURES", errProductTypeEmpty, true, true, true)
}

func TestGetFuturesSubaccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetFuturesSubaccountAssets, "", "COIN-FUTURES", errProductTypeEmpty, true, true, true)
}

func TestGetUSDTInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetUSDTInterestHistory(t.Context(), testFiat, "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetUSDTInterestHistory(t.Context(), testFiat, "woof", 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// This endpoint persistently returns the error "Parameter verification failed" for no discernible reason
	_, err = bi.GetUSDTInterestHistory(t.Context(), testFiat, "SUSDT-FUTURES", 1<<62, 2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetEstimatedOpenCount(t *testing.T) {
	t.Parallel()
	_, err := bi.GetEstimatedOpenCount(t.Context(), currency.Pair{}, "", currency.Code{}, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetEstimatedOpenCount(t.Context(), testPair, "", currency.Code{}, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetEstimatedOpenCount(t.Context(), testPair, "woof", currency.Code{}, 0, 0, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.GetEstimatedOpenCount(t.Context(), testPair, "woof", testFiat, 0, 0, 0)
	assert.ErrorIs(t, err, errOpenAmountEmpty)
	_, err = bi.GetEstimatedOpenCount(t.Context(), testPair, "woof", testFiat, 1, 0, 0)
	assert.ErrorIs(t, err, errOpenPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetEstimatedOpenCount(t.Context(), testPair, "USDT-FUTURES", testFiat, testPrice, testAmount, 20)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestSetIsolatedAutoMargin(t *testing.T) {
	t.Parallel()
	_, err := bi.SetIsolatedAutoMargin(t.Context(), currency.Pair{}, false, currency.Code{}, "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.SetIsolatedAutoMargin(t.Context(), testPair, false, currency.Code{}, "")
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.SetIsolatedAutoMargin(t.Context(), testPair, false, testFiat, "")
	assert.ErrorIs(t, err, errHoldSideEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.SetIsolatedAutoMargin(t.Context(), testPair, false, testFiat, "short")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestChangeLeverage(t *testing.T) {
	t.Parallel()
	_, err := bi.ChangeLeverage(t.Context(), currency.Pair{}, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.ChangeLeverage(t.Context(), testPair, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ChangeLeverage(t.Context(), testPair, "woof", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.ChangeLeverage(t.Context(), testPair, "woof", "", testFiat, 0)
	assert.ErrorIs(t, err, errLeverageEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ChangeLeverage(t.Context(), testPair, "USDT-FUTURES", "", testFiat, 20)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestAdjustMargin(t *testing.T) {
	t.Parallel()
	err := bi.AdjustMargin(t.Context(), currency.Pair{}, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	err = bi.AdjustMargin(t.Context(), testPair2, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	err = bi.AdjustMargin(t.Context(), testPair2, "woof", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	err = bi.AdjustMargin(t.Context(), testPair2, "woof", "", testFiat2, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	err = bi.AdjustMargin(t.Context(), testPair2, "woof", "", testFiat2, 1)
	assert.ErrorIs(t, err, errHoldSideEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	// This is getting the error "verification exception margin mode == FIXED", and I can't find a way to skirt around that
	// Now it's giving the error "insufficient amount of margin", which is a fine error to have, watch for random reversions
	// And back to the former error
	err = bi.AdjustMargin(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "long", testFiat2, -testAmount)
	assert.NoError(t, err)
}

func TestSetUSDTAssetMode(t *testing.T) {
	t.Parallel()
	_, err := bi.SetUSDTAssetMode(t.Context(), "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.SetUSDTAssetMode(t.Context(), "meow", "")
	assert.ErrorIs(t, err, errAssetModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.SetUSDTAssetMode(t.Context(), "SUSDT-FUTURES", "single")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestChangeMarginMode(t *testing.T) {
	t.Parallel()
	_, err := bi.ChangeMarginMode(t.Context(), currency.Pair{}, "", "", currency.Code{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.ChangeMarginMode(t.Context(), testPair2, "", "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ChangeMarginMode(t.Context(), testPair2, "woof", "", currency.Code{})
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.ChangeMarginMode(t.Context(), testPair2, "woof", "", testFiat2)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.ChangeMarginMode(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "crossed", testFiat2)
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
	_, err := bi.ChangePositionMode(t.Context(), "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ChangePositionMode(t.Context(), "meow", "")
	assert.ErrorIs(t, err, errPositionModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ChangePositionMode(t.Context(), testFiat2.String()+"-FUTURES", "hedge_mode")
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesAccountBills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesAccountBills(t.Context(), "", "", "", currency.Code{}, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesAccountBills(t.Context(), "meow", "", "", currency.Code{}, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesAccountBills(t.Context(), testFiat2.String()+"-FUTURES", "trans_from_exchange", "", testFiat2, 1<<62, 2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetPositionTier(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPositionTier(t.Context(), "", currency.Pair{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetPositionTier(t.Context(), "meow", currency.Pair{})
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetPositionTier(t.Context(), testFiat2.String()+"-FUTURES", testPair2)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSinglePosition(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSinglePosition(t.Context(), "", currency.Pair{}, currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetSinglePosition(t.Context(), "meow", currency.Pair{}, currency.Code{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSinglePosition(t.Context(), "meow", testPair2, currency.Code{})
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSinglePosition(t.Context(), testFiat2.String()+"-FUTURES", testPair2, testFiat2)
	assert.NoError(t, err)
}

func TestGetAllPositions(t *testing.T) {
	t.Parallel()
	_, err := bi.GetAllPositions(t.Context(), "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetAllPositions(t.Context(), "meow", currency.Code{})
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetAllPositions(t.Context(), testFiat2.String()+"-FUTURES", testFiat2)
	assert.NoError(t, err)
}

func TestGetHistoricalPositions(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalPositions(t.Context(), currency.Pair{}, "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeAndPairEmpty)
	_, err = bi.GetHistoricalPositions(t.Context(), testPair2, "", 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetHistoricalPositions(t.Context(), testPair2, "", 1<<62, 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceFuturesOrder(t.Context(), currency.Pair{}, "", "", "", "", "", "", "", "", currency.Code{}, 0, 0, 0, 0, false, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceFuturesOrder(t.Context(), testPair2, "", "", "", "", "", "", "", "", currency.Code{}, 0, 0, 0, 0, false, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceFuturesOrder(t.Context(), testPair2, "woof", "", "", "", "", "", "", "", currency.Code{}, 0, 0, 0, 0, false, false)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	_, err = bi.PlaceFuturesOrder(t.Context(), testPair2, "woof", "neigh", "", "", "", "", "", "", currency.Code{}, 0, 0, 0, 0, false, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceFuturesOrder(t.Context(), testPair2, "woof", "neigh", "", "", "", "", "", "", testFiat2, 0, 0, 0, 0, false, false)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceFuturesOrder(t.Context(), testPair2, "woof", "neigh", "oink", "", "", "", "", "", testFiat2, 0, 0, 0, 0, false, false)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceFuturesOrder(t.Context(), testPair2, "woof", "neigh", "oink", "", "limit", "", "", "", testFiat2, 0, 0, 0, 0, false, false)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.PlaceFuturesOrder(t.Context(), testPair2, "woof", "neigh", "oink", "", "limit", "", "", "", testFiat2, 0, 0, 1, 0, false, false)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.PlaceFuturesOrder(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "isolated", "buy", "open", "limit", "GTC", clientIDGenerator(), "", testFiat2, testPrice2+1, testPrice2-1, testAmount2, testPrice2, true, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestPlaceReversal(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceReversal(t.Context(), currency.Pair{}, currency.Code{}, "", "", "", "", 0, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceReversal(t.Context(), testPair2, currency.Code{}, "", "", "", "", 0, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceReversal(t.Context(), testPair2, testFiat2, "", "", "", "", 0, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceReversal(t.Context(), testPair2, testFiat2, "neigh", "", "", "", 0, false)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceReversal(t.Context(), testPair, testFiat, testFiat.String()+"-FUTURES", "Buy", "Open", clientIDGenerator(), 30, true)
	assert.NoError(t, err)
}

func TestBatchPlaceFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.BatchPlaceFuturesOrders(t.Context(), currency.Pair{}, "", "", currency.Code{}, nil, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchPlaceFuturesOrders(t.Context(), testPair2, "", "", currency.Code{}, nil, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.BatchPlaceFuturesOrders(t.Context(), testPair2, "woof", "", currency.Code{}, nil, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.BatchPlaceFuturesOrders(t.Context(), testPair2, "woof", "", testFiat2, nil, false)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	_, err = bi.BatchPlaceFuturesOrders(t.Context(), testPair2, "woof", "neigh", testFiat2, nil, false)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	orders := []PlaceFuturesOrderStruct{
		{
			Size:      testAmount,
			Price:     testPrice,
			Side:      "Sell",
			TradeSide: "Open",
			OrderType: "limit",
			Strategy:  "FOK",
		},
	}
	_, err = bi.BatchPlaceFuturesOrders(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "isolated", testFiat2, orders, true)
	assert.NoError(t, err)
}

func TestGetFuturesOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesOrderDetails(t.Context(), currency.Pair{}, "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesOrderDetails(t.Context(), testPair2, "", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesOrderDetails(t.Context(), testPair2, "woof", "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesOrderDetails(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "a", 1)
	assert.NoError(t, err)
}

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesFills(t.Context(), 0, 0, 0, currency.Pair{}, "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesFills(t.Context(), 0, 0, 0, currency.Pair{}, "meow", time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesFills(t.Context(), 0, 1<<62, 5, currency.Pair{}, testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesOrderFillHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesOrderFillHistory(t.Context(), currency.Pair{}, "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesOrderFillHistory(t.Context(), currency.Pair{}, "meow", 0, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// Keeps getting "Parameter verification failed" error and I can't figure out why
	resp, err := bi.GetFuturesOrderFillHistory(t.Context(), currency.Pair{}, testFiat2.String()+"-FUTURES", 0, 1<<62, 5, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.FillList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetFuturesOrderFillHistory(t.Context(), currency.Pair{}, testFiat2.String()+"-FUTURES", resp.FillList[0].OrderID, 1<<62, 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetPendingFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPendingFuturesOrders(t.Context(), 0, 0, 0, "", "", "", currency.Pair{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetPendingFuturesOrders(t.Context(), 0, 0, 0, "", "meow", "", currency.Pair{}, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetPendingFuturesOrders(t.Context(), 0, 1<<62, 5, "", testFiat2.String()+"-FUTURES", "", testPair2, time.Now().Add(-time.Hour*24*90), time.Now())
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetPendingFuturesOrders(t.Context(), resp.EntrustedList[0].OrderID, 1<<62, 5, "", testFiat2.String()+"-FUTURES", "", testPair2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetHistoricalFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalFuturesOrders(t.Context(), 0, 0, 0, "", "", "", currency.Pair{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetHistoricalFuturesOrders(t.Context(), 0, 0, 0, "", "meow", "", currency.Pair{}, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetHistoricalFuturesOrders(t.Context(), 0, 1<<62, 5, "", testFiat2.String()+"-FUTURES", "", testPair2, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetHistoricalFuturesOrders(t.Context(), resp.EntrustedList[0].OrderID, 1<<62, 5, "", testFiat2.String()+"-FUTURES", "", testPair2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesTriggerOrderByID(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesTriggerOrderByID(t.Context(), "", "", 0)
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.GetFuturesTriggerOrderByID(t.Context(), "meow", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesTriggerOrderByID(t.Context(), "meow", "woof", 0)
	assert.ErrorIs(t, err, errPlanOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 1<<62, 5, "", "normal_plan", "", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetFuturesTriggerOrderByID(t.Context(), "normal_plan", testFiat2.String()+"-FUTURES", resp.EntrustedList[0].OrderID)
	assert.NoError(t, err)
}

func TestPlaceTPSLFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceTPSLFuturesOrder(t.Context(), currency.Code{}, "", "", "", "", "", "", "", currency.Pair{}, 0, 0, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(t.Context(), testFiat2, "", "", "", "", "", "", "", currency.Pair{}, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(t.Context(), testFiat2, "woof", "", "", "", "", "", "", currency.Pair{}, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(t.Context(), testFiat2, "woof", "", "", "", "", "", "", testPair2, 0, 0, 0)
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(t.Context(), testFiat2, "woof", "neigh", "", "", "", "", "", testPair2, 0, 0, 0)
	assert.ErrorIs(t, err, errHoldSideEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(t.Context(), testFiat2, "woof", "neigh", "", "quack", "", "", "", testPair2, 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(t.Context(), testFiat2, "woof", "neigh", "", "quack", "", "", "", testPair2, 1, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	cID := clientIDGenerator()
	resp, err := bi.PlaceTPSLFuturesOrder(t.Context(), testFiat2, testFiat2.String()+"-FUTURES", "profit_plan", "", "short", "", cID, "", testPair2, testPrice2+2, 0, testAmount2)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestPlaceTPAndSLFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceTPAndSLFuturesOrder(t.Context(), currency.Code{}, "", "", "", "", "", currency.Pair{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceTPAndSLFuturesOrder(t.Context(), testFiat2, "", "", "", "", "", currency.Pair{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceTPAndSLFuturesOrder(t.Context(), testFiat2, "woof", "", "", "", "", currency.Pair{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceTPAndSLFuturesOrder(t.Context(), testFiat2, "woof", "", "", "", "", testPair2, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errTakeProfitTriggerPriceEmpty)
	_, err = bi.PlaceTPAndSLFuturesOrder(t.Context(), testFiat2, "woof", "", "", "", "", testPair2, 1, 0, 0, 0)
	assert.ErrorIs(t, err, errStopLossTriggerPriceEmpty)
	_, err = bi.PlaceTPAndSLFuturesOrder(t.Context(), testFiat2, "woof", "", "", "", "", testPair2, 1, 0, 1, 0)
	assert.ErrorIs(t, err, errHoldSideEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.PlaceTPAndSLFuturesOrder(t.Context(), testFiat2, testFiat2.String()+"-FUTURES", "fill_price", "fill_price", "short", "", testPair2, 2*testPrice2+1, 2*testPrice2, testPrice2-1, testPrice2)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestPlaceTriggerFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceTriggerFuturesOrder(t.Context(), "", "", "", "", "", "", "", "", "", "", "", currency.Pair{}, currency.Code{}, 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "", "", "", "", "", "", "", "", "", "", currency.Pair{}, currency.Code{}, 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "", "", "", "", "", "", "", "", "", "", testPair2, currency.Code{}, 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "woof", "", "", "", "", "", "", "", "", "", testPair2, currency.Code{}, 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "woof", "neigh", "", "", "", "", "", "", "", "", testPair2, currency.Code{}, 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "woof", "neigh", "", "", "", "", "", "", "", "", testPair2, testFiat2, 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errTriggerTypeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "woof", "neigh", "oink", "", "", "", "", "", "", "", testPair2, testFiat2, 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "woof", "neigh", "oink", "quack", "", "", "", "", "", "", testPair2, testFiat2, 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "woof", "neigh", "oink", "quack", "", "cluck", "", "", "", "", testPair2, testFiat2, 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "woof", "neigh", "oink", "quack", "", "cluck", "", "", "", "", testPair2, testFiat2, 1, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errExecutePriceEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(t.Context(), "meow", "woof", "neigh", "oink", "quack", "", "cluck", "", "", "", "", testPair2, testFiat2, 1, 1, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	// This returns the error "The parameter does not meet the specification d delegateType is error". The documentation doesn't mention that parameter anywhere, nothing seems similar to it, and attempts to send that parameter with various values, or to tweak other parameters, yielded no difference
	resp, err := bi.PlaceTriggerFuturesOrder(t.Context(), "normal_plan", testFiat2.String()+"-FUTURES", "isolated", "mark_price", "Sell", "", "limit", clientIDGenerator(), "fill_price", "fill_price", "", testPair2, testFiat2, testAmount2*1000, testPrice2+2, 0, testPrice2+1, testPrice2+3, testPrice2-2, testPrice2-1, testPrice2/2, true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyTPSLFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.ModifyTPSLFuturesOrder(t.Context(), 0, "", "", "", currency.Code{}, currency.Pair{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(t.Context(), 1, "", "", "", currency.Code{}, currency.Pair{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(t.Context(), 1, "", "", "", testFiat2, currency.Pair{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(t.Context(), 1, "", "meow", "", testFiat2, currency.Pair{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(t.Context(), 1, "", "meow", "", testFiat2, testPair2, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(t.Context(), 1, "", "meow", "", testFiat2, testPair2, 1, 0, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ModifyTPSLFuturesOrder(t.Context(), 1, "a", testFiat2.String()+"-FUTURES", "", testFiat2, testPair2, testPrice2-1, testPrice2+2, testAmount2, 0.1)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyTriggerFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.ModifyTriggerFuturesOrder(t.Context(), 0, "", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.ModifyTriggerFuturesOrder(t.Context(), 1, "", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	t.Skip("TODO: Finish once PlaceTriggerFuturesOrder is fixed")
}

func TestGetPendingTriggerFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPendingTriggerFuturesOrders(t.Context(), 0, 0, 0, "", "", "", currency.Pair{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.GetPendingTriggerFuturesOrders(t.Context(), 0, 0, 0, "", "meow", "", currency.Pair{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetPendingTriggerFuturesOrders(t.Context(), 0, 0, 0, "", "meow", "woof", currency.Pair{}, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetPendingTriggerFuturesOrders(t.Context(), 0, 1<<62, 5, "", "profit_loss", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetPendingTriggerFuturesOrders(t.Context(), resp.EntrustedList[0].OrderID, 1<<62, 5, "", "profit_loss", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetHistoricalTriggerFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 0, 0, "", "", "", "", testPair2, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 0, 0, "", "meow", "", "", testPair2, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 0, 0, "", "meow", "", "woof", testPair2, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetHistoricalTriggerFuturesOrders(t.Context(), 0, 1<<62, 5, "", "normal_plan", "", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetHistoricalTriggerFuturesOrders(t.Context(), resp.EntrustedList[0].OrderID, 1<<62, 5, "", "normal_plan", "", testFiat2.String()+"-FUTURES", testPair2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetSupportedCurrencies(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetSupportedCurrencies)
}

func TestGetCrossBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossBorrowHistory(t.Context(), 0, 0, 0, currency.Code{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossBorrowHistory(t.Context(), 1, 2, 1<<62, testFiat, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossRepayHistory(t.Context(), 0, 0, 0, currency.Code{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossRepayHistory(t.Context(), 1, 2, 1<<62, testFiat, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossInterestHistory(t.Context(), currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossInterestHistory(t.Context(), testFiat, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetCrossLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossLiquidationHistory(t.Context(), time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossLiquidationHistory(t.Context(), time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetCrossFinancialHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossFinancialHistory(t.Context(), "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossFinancialHistory(t.Context(), "", testFiat, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetCrossAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossAccountAssets, currency.Code{}, testFiat, nil, false, true, true)
}

func TestCrossBorrow(t *testing.T) {
	t.Parallel()
	_, err := bi.CrossBorrow(t.Context(), currency.Code{}, "", 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.CrossBorrow(t.Context(), testFiat, "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CrossBorrow(t.Context(), testFiat, clientIDGenerator(), testAmount)
	assert.NoError(t, err)
}

func TestCrossRepay(t *testing.T) {
	t.Parallel()
	_, err := bi.CrossRepay(t.Context(), currency.Code{}, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.CrossRepay(t.Context(), testFiat, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CrossRepay(t.Context(), testFiat, testAmount)
	assert.NoError(t, err)
}

func TestGetCrossRiskRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetCrossRiskRate)
}

func TestGetCrossMaxBorrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossMaxBorrowable, currency.Code{}, testFiat, errCurrencyEmpty, false, true, true)
}

func TestGetCrossMaxTransferable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossMaxTransferable, currency.Code{}, testFiat, errCurrencyEmpty, false, true, true)
}

func TestGetCrossInterestRateAndMaxBorrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossInterestRateAndMaxBorrowable, currency.Code{}, testFiat, errCurrencyEmpty, false, true, true)
}

func TestGetCrossTierConfiguration(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossTierConfiguration, currency.Code{}, testFiat, errCurrencyEmpty, false, true, true)
}

func TestCrossFlashRepay(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.CrossFlashRepay, currency.Code{}, testFiat, nil, false, true, canManipulateRealOrders)
}

func TestGetCrossFlashRepayResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossFlashRepayResult(t.Context(), nil)
	assert.ErrorIs(t, err, errIDListEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	// This must be done, as this is the only way to get a repayment ID
	resp, err := bi.CrossFlashRepay(t.Context(), testFiat)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = bi.GetCrossFlashRepayResult(t.Context(), []int64{resp.RepayID})
	assert.NoError(t, err)
}

func TestPlaceCrossOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceCrossOrder(t.Context(), currency.Pair{}, "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceCrossOrder(t.Context(), testPair, "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceCrossOrder(t.Context(), testPair, "woof", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errLoanTypeEmpty)
	_, err = bi.PlaceCrossOrder(t.Context(), testPair, "woof", "neigh", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errStrategyEmpty)
	_, err = bi.PlaceCrossOrder(t.Context(), testPair, "woof", "neigh", "oink", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceCrossOrder(t.Context(), testPair, "woof", "neigh", "oink", "", "quack", "", 0, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceCrossOrder(t.Context(), testPair, "limit", "normal", "GTC", "", "sell", "", testPrice, testAmount, testAmount*testPrice)
	assert.NoError(t, err)
}

func TestBatchPlaceCrossOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.BatchPlaceCrossOrders(t.Context(), currency.Pair{}, nil)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchPlaceCrossOrders(t.Context(), testPair, nil)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
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
	_, err = bi.BatchPlaceCrossOrders(t.Context(), testPair, orders)
	assert.NoError(t, err)
}

func TestGetCrossOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossOpenOrders(t.Context(), currency.Pair{}, "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCrossOpenOrders(t.Context(), testPair, "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossOpenOrders(t.Context(), testPair, "", 1, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossHistoricalorders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossHistoricalOrders(t.Context(), currency.Pair{}, "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCrossHistoricalOrders(t.Context(), testPair, "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetCrossHistoricalOrders(t.Context(), testPair, "", "", 0, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetCrossHistoricalOrders(t.Context(), testPair, "", "", resp.OrderList[0].OrderID, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossOrderFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossOrderFills(t.Context(), currency.Pair{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCrossOrderFills(t.Context(), testPair, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetCrossOrderFills(t.Context(), testPair, 0, 1<<62, 5, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Fills) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetCrossOrderFills(t.Context(), testPair, resp.Fills[0].OrderID, 1<<62, 5, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossLiquidationOrders(t.Context(), "", "", "", currency.Pair{}, time.Now().Add(time.Hour), time.Time{}, 5, 1<<62)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossLiquidationOrders(t.Context(), "swap", "", "", currency.Pair{}, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedRepayHistory(t.Context(), currency.Pair{}, currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedRepayHistory(t.Context(), testPair, currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetIsolatedRepayHistory(t.Context(), testPair, currency.Code{}, 0, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.ResultList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetIsolatedRepayHistory(t.Context(), testPair, testFiat, resp.ResultList[0].RepayID, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedBorrowHistory(t.Context(), currency.Pair{}, currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedBorrowHistory(t.Context(), testPair, currency.Code{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedBorrowHistory(t.Context(), testPair, currency.Code{}, 0, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedInterestHistory(t.Context(), currency.Pair{}, currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedInterestHistory(t.Context(), testPair, currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedInterestHistory(t.Context(), testPair, testFiat, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedLiquidationHistory(t.Context(), currency.Pair{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedLiquidationHistory(t.Context(), testPair, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedLiquidationHistory(t.Context(), testPair, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedFinancialHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedFinancialHistory(t.Context(), currency.Pair{}, "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedFinancialHistory(t.Context(), testPair, "", currency.Code{}, time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedFinancialHistory(t.Context(), testPair, "", testCrypto, time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetIsolatedAccountAssets, currency.Pair{}, currency.Pair{}, nil, false, true, true)
}

func TestIsolatedBorrow(t *testing.T) {
	t.Parallel()
	_, err := bi.IsolatedBorrow(t.Context(), currency.Pair{}, currency.Code{}, "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.IsolatedBorrow(t.Context(), testPair, currency.Code{}, "", 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.IsolatedBorrow(t.Context(), testPair, testFiat, "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.IsolatedBorrow(t.Context(), testPair, testFiat, "", testAmount)
	assert.NoError(t, err)
}

func TestIsolatedRepay(t *testing.T) {
	t.Parallel()
	_, err := bi.IsolatedRepay(t.Context(), 0, currency.Code{}, currency.Pair{}, "")
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.IsolatedRepay(t.Context(), 1, currency.Code{}, currency.Pair{}, "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.IsolatedRepay(t.Context(), 1, testFiat, currency.Pair{}, "")
	assert.ErrorIs(t, err, errPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.IsolatedRepay(t.Context(), testAmount, testFiat, testPair, "")
	assert.NoError(t, err)
}

func TestGetIsolatedRiskRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetIsolatedRiskRate(t.Context(), currency.Pair{}, 1, 5)
	assert.NoError(t, err)
}

func TestGetIsolatedInterestRateAndMaxBorrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetIsolatedInterestRateAndMaxBorrowable, currency.Pair{}, testPair, errPairEmpty, false, true, true)
}

func TestGetIsolatedTierConfiguration(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetIsolatedTierConfiguration, currency.Pair{}, testPair, errPairEmpty, false, true, true)
}

func TestGetIsolatedMaxborrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetIsolatedMaxBorrowable, currency.Pair{}, testPair, errPairEmpty, false, true, true)
}

func TestGetIsolatedMaxTransferable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetIsolatedMaxTransferable, currency.Pair{}, testPair, errPairEmpty, false, true, true)
}

func TestIsolatedFlashRepay(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.IsolatedFlashRepay, nil, currency.Pairs{testPair}, nil, false, true, canManipulateRealOrders)
}

func TestGetIsolatedFlashRepayResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedFlashRepayResult(t.Context(), nil)
	assert.ErrorIs(t, err, errIDListEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	// This must be done, as this is the only way to get a repayment ID
	resp, err := bi.IsolatedFlashRepay(t.Context(), currency.Pairs{testPair})
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = bi.GetIsolatedFlashRepayResult(t.Context(), []int64{resp[0].RepayID})
	assert.NoError(t, err)
}

func TestPlaceIsolatedOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceIsolatedOrder(t.Context(), currency.Pair{}, "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceIsolatedOrder(t.Context(), testPair, "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceIsolatedOrder(t.Context(), testPair, "meow", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errLoanTypeEmpty)
	_, err = bi.PlaceIsolatedOrder(t.Context(), testPair, "meow", "woof", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errStrategyEmpty)
	_, err = bi.PlaceIsolatedOrder(t.Context(), testPair, "meow", "woof", "neigh", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceIsolatedOrder(t.Context(), testPair, "meow", "woof", "neigh", "", "quack", "", 0, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceIsolatedOrder(t.Context(), testPair, "limit", "normal", "GTC", "", "sell", "", testPrice, testAmount, testPrice*testAmount)
	assert.NoError(t, err)
}

func TestBatchPlaceIsolatedOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.BatchPlaceIsolatedOrders(t.Context(), currency.Pair{}, nil)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchPlaceIsolatedOrders(t.Context(), testPair, nil)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
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
	_, err = bi.BatchPlaceIsolatedOrders(t.Context(), testPair, orders)
	assert.NoError(t, err)
}

func TestGetIsolatedOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedOpenOrders(t.Context(), currency.Pair{}, "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedOpenOrders(t.Context(), testPair, "", 0, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedOpenOrders(t.Context(), testPair, "", 1, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedHistoricalOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedHistoricalOrders(t.Context(), currency.Pair{}, "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedHistoricalOrders(t.Context(), testPair, "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetIsolatedHistoricalOrders(t.Context(), testPair, "", "", 0, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetIsolatedHistoricalOrders(t.Context(), testPair, "", "", resp.OrderList[0].OrderID, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedOrderFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedOrderFills(t.Context(), currency.Pair{}, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedOrderFills(t.Context(), testPair, 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetIsolatedOrderFills(t.Context(), testPair, 0, 1<<62, 5, time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Fills) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetIsolatedOrderFills(t.Context(), testPair, resp.Fills[0].OrderID, 1<<62, 5, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedLiquidationOrders(t.Context(), "", "", "", currency.Pair{}, time.Now().Add(time.Hour), time.Time{}, 5, 1<<62)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedLiquidationOrders(t.Context(), "place_order", "", "", currency.Pair{}, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSavingsProductList(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsProductList(t.Context(), currency.Code{}, "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSavingsProductList(t.Context(), testCrypto, "")
	assert.NoError(t, err)
}

func TestGetSavingsBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetSavingsBalance)
}

func TestGetSavingsAssets(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsAssets(t.Context(), "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSavingsAssets(t.Context(), "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSavingsRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsRecords(t.Context(), currency.Code{}, "", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSavingsRecords(t.Context(), currency.Code{}, "", "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSavingsSubscriptionDetail(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsSubscriptionDetail(t.Context(), 0, "")
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = bi.GetSavingsSubscriptionDetail(t.Context(), 1, "")
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSavingsProductList(t.Context(), testCrypto, "")
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = bi.GetSavingsSubscriptionDetail(t.Context(), resp[0].ProductID, resp[0].PeriodType)
	assert.NoError(t, err)
}

func TestSubscribeSavings(t *testing.T) {
	t.Parallel()
	_, err := bi.SubscribeSavings(t.Context(), 0, "", 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = bi.SubscribeSavings(t.Context(), 1, "", 0)
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	_, err = bi.SubscribeSavings(t.Context(), 1, "meow", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetSavingsProductList(t.Context(), testCrypto, "")
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	resp2, err := bi.GetSavingsSubscriptionDetail(t.Context(), resp[0].ProductID, resp[0].PeriodType)
	require.NoError(t, err)
	require.NotEmpty(t, resp2)
	_, err = bi.SubscribeSavings(t.Context(), resp[0].ProductID, resp[0].PeriodType, resp2.SingleMinAmount)
	assert.NoError(t, err)
}

func TestGetSavingsSubscriptionResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsSubscriptionResult(t.Context(), 0, "")
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = bi.GetSavingsSubscriptionResult(t.Context(), 1, "")
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSavingsRecords(t.Context(), currency.Code{}, "", "", time.Time{}, time.Time{}, 100, 0)
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
	_, err = bi.GetSavingsSubscriptionResult(t.Context(), resp.ResultList[tarID].OrderID, resp.ResultList[tarID].ProductType)
	assert.NoError(t, err)
}

func TestRedeemSavings(t *testing.T) {
	t.Parallel()
	_, err := bi.RedeemSavings(t.Context(), 0, 0, "", 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = bi.RedeemSavings(t.Context(), 1, 0, "", 0)
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	_, err = bi.RedeemSavings(t.Context(), 1, 0, "meow", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	var pagination int64
	var tarProd int64
	var tarOrd int64
	var tarPeriod string
	for {
		resp, err := bi.GetSavingsAssets(t.Context(), "", time.Time{}, time.Time{}, 100, pagination)
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
		if tarProd != 0 || int64(resp.EndID) == pagination || resp.EndID == 0 {
			break
		}
		pagination = int64(resp.EndID)
	}
	if tarProd == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.RedeemSavings(t.Context(), tarProd, tarOrd, tarPeriod, testAmount)
	assert.NoError(t, err)
}

func TestGetSavingsRedemptionResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsRedemptionResult(t.Context(), 0, "")
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = bi.GetSavingsRedemptionResult(t.Context(), 1, "")
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSavingsRecords(t.Context(), currency.Code{}, "", "", time.Time{}, time.Time{}, 100, 0)
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
	_, err = bi.GetSavingsRedemptionResult(t.Context(), resp.ResultList[tarID].OrderID, resp.ResultList[tarID].ProductType)
	assert.NoError(t, err)
}

func TestGetEarnAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetEarnAccountAssets, currency.Code{}, currency.Code{}, nil, false, true, true)
}

func TestGetSharkFinProducts(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinProducts(t.Context(), currency.Code{}, 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSharkFinProducts(t.Context(), testCrypto, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSharkFinBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetSharkFinBalance)
}

func TestGetSharkFinAssets(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinAssets(t.Context(), "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSharkFinAssets(t.Context(), "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSharkFinRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinRecords(t.Context(), currency.Code{}, "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSharkFinRecords(t.Context(), currency.Code{}, "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSharkFinSubscriptionDetail(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinSubscriptionDetail(t.Context(), 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSharkFinProducts(t.Context(), testCrypto, 5, 1<<62)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = bi.GetSharkFinSubscriptionDetail(t.Context(), resp.ResultList[0].ProductID)
	assert.NoError(t, err)
}

func TestSubscribeSharkFin(t *testing.T) {
	t.Parallel()
	_, err := bi.SubscribeSharkFin(t.Context(), 0, 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = bi.SubscribeSharkFin(t.Context(), 1, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetSharkFinProducts(t.Context(), testCrypto, 5, 1<<62)
	require.NoError(t, err)
	require.NotEmpty(t, resp)
	_, err = bi.SubscribeSharkFin(t.Context(), resp.ResultList[0].ProductID, resp.ResultList[0].MinimumAmount)
	assert.NoError(t, err)
}

func TestGetSharkFinSubscriptionResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinSubscriptionResult(t.Context(), 0)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSharkFinRecords(t.Context(), currency.Code{}, "", time.Time{}, time.Time{}, 100, 0)
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
	_, err = bi.GetSharkFinSubscriptionResult(t.Context(), resp[tarID].OrderID)
	assert.NoError(t, err)
}

func TestGetLoanCurrencyList(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetLoanCurrencyList, currency.Code{}, testFiat, errCurrencyEmpty, false, true, true)
}

func TestGetEstimatedInterestAndBorrowable(t *testing.T) {
	t.Parallel()
	_, err := bi.GetEstimatedInterestAndBorrowable(t.Context(), currency.Code{}, currency.Code{}, "", 0)
	assert.ErrorIs(t, err, errLoanCoinEmpty)
	_, err = bi.GetEstimatedInterestAndBorrowable(t.Context(), testCrypto, currency.Code{}, "", 0)
	assert.ErrorIs(t, err, errCollateralCoinEmpty)
	_, err = bi.GetEstimatedInterestAndBorrowable(t.Context(), testCrypto, testFiat, "", 0)
	assert.ErrorIs(t, err, errTermEmpty)
	_, err = bi.GetEstimatedInterestAndBorrowable(t.Context(), testCrypto, testFiat, "neigh", 0)
	assert.ErrorIs(t, err, errCollateralAmountEmpty)
	_, err = bi.GetEstimatedInterestAndBorrowable(t.Context(), testCrypto, testFiat, "SEVEN", testPrice)
	assert.NoError(t, err)
}

func TestBorrowFunds(t *testing.T) {
	t.Parallel()
	_, err := bi.BorrowFunds(t.Context(), currency.Code{}, currency.Code{}, "", 0, 0)
	assert.ErrorIs(t, err, errLoanCoinEmpty)
	_, err = bi.BorrowFunds(t.Context(), testCrypto, currency.Code{}, "", 0, 0)
	assert.ErrorIs(t, err, errCollateralCoinEmpty)
	_, err = bi.BorrowFunds(t.Context(), testCrypto, testFiat, "", 0, 0)
	assert.ErrorIs(t, err, errTermEmpty)
	_, err = bi.BorrowFunds(t.Context(), testCrypto, testFiat, "neigh", 0, 0)
	assert.ErrorIs(t, err, errCollateralLoanMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.BorrowFunds(t.Context(), testCrypto, testFiat, "SEVEN", testAmount, 0)
	assert.NoError(t, err)
}

func TestGetOngoingLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetOngoingLoans(t.Context(), 0, currency.Code{}, currency.Code{})
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetOngoingLoans(t.Context(), resp[0].OrderID, currency.Code{}, currency.Code{})
	assert.NoError(t, err)
}

func TestGetLoanRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetLoanRepayHistory(t.Context(), 0, 0, 0, currency.Code{}, currency.Code{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetLoanRepayHistory(t.Context(), 0, 1, 5, currency.Code{}, currency.Code{}, time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestModifyPledgeRate(t *testing.T) {
	t.Parallel()
	_, err := bi.ModifyPledgeRate(t.Context(), 0, 0, currency.Code{}, "")
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = bi.ModifyPledgeRate(t.Context(), 1, 0, currency.Code{}, "")
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.ModifyPledgeRate(t.Context(), 1, 1, currency.Code{}, "")
	assert.ErrorIs(t, err, errCollateralCoinEmpty)
	_, err = bi.ModifyPledgeRate(t.Context(), 1, 1, currency.NewCode("meow"), "")
	assert.ErrorIs(t, err, errReviseTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetOngoingLoans(t.Context(), 0, currency.Code{}, currency.Code{})
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.ModifyPledgeRate(t.Context(), resp[0].OrderID, testAmount, testFiat, "IN")
	assert.NoError(t, err)
}

func TestGetPledgeRateHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPledgeRateHistory(t.Context(), 0, 0, 0, "", currency.Code{}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetPledgeRateHistory(t.Context(), 0, 1, 5, "", currency.Code{}, time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestGetLoanHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetLoanHistory(t.Context(), 0, 0, 0, currency.Code{}, currency.Code{}, "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetLoanHistory(t.Context(), 0, 1, 5, currency.Code{}, currency.Code{}, "", time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestGetDebts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// If there aren't any debts to return information on, this will return the error "The data fetched by {user ID} is empty"
	testGetNoArgs(t, bi.GetDebts)
}

func TestGetLiquidationRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetLiquidationRecords(t.Context(), 0, 0, 0, currency.Code{}, currency.Code{}, "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetLiquidationRecords(t.Context(), 0, 1, 5, currency.Code{}, currency.Code{}, "", time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestGetLoanInfo(t *testing.T) {
	t.Parallel()
	t.Skip(skipInstitution)
	testGetOneArg(t, bi.GetLoanInfo, "", "1", errProductIDEmpty, false, true, true)
}

func TestGetMarginCoinRatio(t *testing.T) {
	t.Parallel()
	t.Skip(skipInstitution)
	testGetOneArg(t, bi.GetMarginCoinRatio, "", "1", errProductIDEmpty, false, true, true)
}

func TestGetSpotSymbols(t *testing.T) {
	t.Parallel()
	t.Skip(skipInstitution)
	testGetOneArg(t, bi.GetSpotSymbols, "", "1", errProductIDEmpty, false, true, true)
}

func TestGetLoanToValue(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := riskUnitHelper(t)
	_, err := bi.GetLoanToValue(t.Context(), tarID)
	assert.NoError(t, err)
}

func TestGetTransferableAmount(t *testing.T) {
	t.Parallel()
	_, err := bi.GetTransferableAmount(t.Context(), "", currency.Code{})
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetTransferableAmount(t.Context(), "", testFiat)
	assert.NoError(t, err)
}

func TestGetRiskUnit(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetRiskUnit)
}

func TestSubaccountRiskUnitBinding(t *testing.T) {
	t.Parallel()
	_, err := bi.SubaccountRiskUnitBinding(t.Context(), "", "", false)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := subAccTestHelper(t, strings.ToLower(testSubaccountName[:3])+"****@virtual-bitget.com", "")
	tarID2 := riskUnitHelper(t)
	_, err = bi.SubaccountRiskUnitBinding(t.Context(), tarID, tarID2, false)
	assert.NoError(t, err)
}

func TestGetLoanOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetLoanOrders(t.Context(), "", time.Now().Add(time.Minute), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetLoanOrders(t.Context(), "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetRepaymentOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetRepaymentOrders(t.Context(), 0, time.Now().Add(time.Minute), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetRepaymentOrders(t.Context(), 10, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.FetchTradablePairs, asset.Empty, asset.Spot, asset.ErrNotSupported, false, false, false)
	testGetOneArg(t, bi.FetchTradablePairs, 0, asset.Futures, nil, false, false, false)
	testGetOneArg(t, bi.FetchTradablePairs, 0, asset.Margin, nil, false, false, false)
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, bi)
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := bi.UpdateTicker(t.Context(), currency.Pair{}, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = bi.UpdateTicker(t.Context(), fakePair, asset.Spot)
	assert.Error(t, err)
	_, err = bi.UpdateTicker(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.UpdateTicker(t.Context(), fakePair, asset.Futures)
	assert.Error(t, err)
	_, err = bi.UpdateTicker(t.Context(), testPair, asset.Futures)
	assert.NoError(t, err)
	_, err = bi.UpdateTicker(t.Context(), fakePair, asset.Margin)
	assert.Error(t, err)
	_, err = bi.UpdateTicker(t.Context(), testPair, asset.Margin)
	assert.NoError(t, err)
	_, err = bi.UpdateTicker(t.Context(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, bi)
	err := bi.UpdateTickers(t.Context(), asset.Spot)
	assert.NoError(t, err)
	err = bi.UpdateTickers(t.Context(), asset.Futures)
	assert.NoError(t, err)
	err = bi.UpdateTickers(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = bi.UpdateTickers(t.Context(), asset.Margin)
	assert.NoError(t, err)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := bi.FetchTicker(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.FetchTicker(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.FetchTicker(t.Context(), fakePair, asset.Spot)
	assert.Error(t, err)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	_, err := bi.FetchOrderbook(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.FetchOrderbook(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.FetchOrderbook(t.Context(), fakePair, asset.Spot)
	assert.Error(t, err)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := bi.UpdateOrderbook(t.Context(), currency.Pair{}, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = bi.UpdateOrderbook(t.Context(), fakePair, asset.Spot)
	assert.Error(t, err)
	_, err = bi.UpdateOrderbook(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.UpdateOrderbook(t.Context(), fakePair, asset.Futures)
	assert.Error(t, err)
	_, err = bi.UpdateOrderbook(t.Context(), testPair, asset.Futures)
	assert.NoError(t, err)
	_, err = bi.UpdateOrderbook(t.Context(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err := bi.UpdateAccountInfo(t.Context(), asset.Spot)
	assert.NoError(t, err)
	_, err = bi.UpdateAccountInfo(t.Context(), asset.Futures)
	assert.NoError(t, err)
	_, err = bi.UpdateAccountInfo(t.Context(), asset.Margin)
	assert.NoError(t, err)
	_, err = bi.UpdateAccountInfo(t.Context(), asset.CrossMargin)
	assert.NoError(t, err)
	_, err = bi.UpdateAccountInfo(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	var fakeBitget Bitget
	_, err := fakeBitget.FetchAccountInfo(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.FetchAccountInfo(t.Context(), asset.Futures)
	assert.NoError(t, err)
	// When called by itself, the first call will update the account info, while the second call will return it from GetHoldings; we want coverage for both code paths
	_, err = bi.FetchAccountInfo(t.Context(), asset.Futures)
	assert.NoError(t, err)
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetAccountFundingHistory)
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetWithdrawalsHistory(t.Context(), testCrypto, 0)
	assert.NoError(t, err)
	_, err = bi.GetWithdrawalsHistory(t.Context(), fakeCurrency, 0)
	assert.Error(t, err)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetRecentTrades(t.Context(), currency.Pair{}, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = bi.GetRecentTrades(t.Context(), fakePair, asset.Spot)
	assert.Error(t, err)
	_, err = bi.GetRecentTrades(t.Context(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.GetRecentTrades(t.Context(), fakePair, asset.Futures)
	assert.Error(t, err)
	_, err = bi.GetRecentTrades(t.Context(), testPair, asset.Futures)
	assert.NoError(t, err)
	_, err = bi.GetRecentTrades(t.Context(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricTrades(t.Context(), currency.Pair{}, asset.Spot, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = bi.GetHistoricTrades(t.Context(), fakePair, asset.Spot, time.Time{}, time.Time{})
	assert.Error(t, err)
	_, err = bi.GetHistoricTrades(t.Context(), testPair, asset.Spot, time.Now().Add(-time.Hour*24*7), time.Now())
	assert.NoError(t, err)
	_, err = bi.GetHistoricTrades(t.Context(), fakePair, asset.Futures, time.Time{}, time.Time{})
	assert.Error(t, err)
	_, err = bi.GetHistoricTrades(t.Context(), testPair, asset.Futures, time.Now().Add(-time.Hour*24*7), time.Now())
	assert.NoError(t, err)
	_, err = bi.GetHistoricTrades(t.Context(), testPair, asset.Empty, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetServerTime, 0, 0, nil, false, false, true)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	var ord *order.Submit
	_, err := bi.SubmitOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrSubmissionIsNil)
	ord = &order.Submit{
		Exchange:          bi.Name,
		Pair:              testPair,
		AssetType:         asset.Binary,
		Side:              order.Sell,
		Type:              order.Limit,
		Amount:            testAmount,
		Price:             testPrice,
		ImmediateOrCancel: true,
		PostOnly:          true,
	}
	_, err = bi.SubmitOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errStrategyMutex)
	ord.PostOnly = false
	_, err = bi.SubmitOrder(t.Context(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ord.AssetType = asset.Futures
	_, err = bi.SubmitOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ord.AssetType = asset.Spot
	_, err = bi.SubmitOrder(t.Context(), ord)
	assert.NoError(t, err)
	ord.AssetType = asset.CrossMargin
	ord.ImmediateOrCancel = false
	ord.Side = order.Buy
	ord.Amount = testAmount2
	ord.Price = testPrice2
	_, err = bi.SubmitOrder(t.Context(), ord)
	assert.NoError(t, err)
	ord.AssetType = asset.Margin
	ord.AutoBorrow = true
	_, err = bi.SubmitOrder(t.Context(), ord)
	assert.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := bi.GetOrderInfo(t.Context(), "", testPair, asset.Empty)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	_, err = bi.GetOrderInfo(t.Context(), "0", testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetOrderInfo(t.Context(), "1", testPair2, asset.Futures)
	assert.NoError(t, err)
	_, err = bi.GetOrderInfo(t.Context(), "2", testPair, asset.Margin)
	assert.NoError(t, err)
	_, err = bi.GetOrderInfo(t.Context(), "3", testPair, asset.CrossMargin)
	assert.NoError(t, err)
	resp, err := bi.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) != 0 {
		_, err = bi.GetOrderInfo(t.Context(), strconv.FormatInt(int64(resp[0].OrderID), 10), testPair, asset.Spot)
		assert.NoError(t, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := bi.GetDepositAddress(t.Context(), currency.NewCode(""), "", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetDepositAddress(t.Context(), testCrypto, "", "")
	assert.NoError(t, err)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	var req *withdraw.Request
	_, err := bi.WithdrawCryptocurrencyFunds(t.Context(), req)
	assert.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	req = &withdraw.Request{
		Crypto: withdraw.CryptoRequest{
			Address: testAddress,
			Chain:   testCrypto.String(),
		},
		Currency: testCrypto,
		Amount:   testAmount,
		Exchange: bi.Name,
	}
	_, err = bi.WithdrawCryptocurrencyFunds(t.Context(), req)
	assert.NoError(t, err)
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.WithdrawFiatFunds, nil, nil, common.ErrFunctionNotSupported, false, true, false)
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := bi.WithdrawFiatFundsToInternationalBank(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var req *order.MultiOrderRequest
	_, err := bi.GetActiveOrders(t.Context(), req)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil)
	req = &order.MultiOrderRequest{
		AssetType: asset.Binary,
		Side:      order.Sell,
		Type:      order.Limit,
		Pairs:     []currency.Pair{testPair},
	}
	_, err = bi.GetActiveOrders(t.Context(), req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	req.AssetType = asset.CrossMargin
	_, err = bi.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Margin
	_, err = bi.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Futures
	_, err = bi.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{}
	_, err = bi.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Spot
	_, err = bi.GetActiveOrders(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	req.Pairs = []currency.Pair{}
	_, err = bi.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{testPair}
	_, err = bi.GetActiveOrders(t.Context(), req)
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var req *order.MultiOrderRequest
	_, err := bi.GetOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil)
	req = &order.MultiOrderRequest{
		AssetType: asset.Binary,
		Side:      order.Sell,
		Type:      order.Limit,
		Pairs:     []currency.Pair{testPair},
	}
	_, err = bi.GetOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	req.AssetType = asset.CrossMargin
	_, err = bi.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Margin
	_, err = bi.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Futures
	_, err = bi.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{}
	_, err = bi.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Spot
	_, err = bi.GetOrderHistory(t.Context(), req)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	req.Pairs = []currency.Pair{}
	_, err = bi.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{testPair}
	_, err = bi.GetOrderHistory(t.Context(), req)
	assert.NoError(t, err)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	var fb *exchange.FeeBuilder
	_, err := bi.GetFeeByType(t.Context(), fb)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	fb = &exchange.FeeBuilder{}
	_, err = bi.GetFeeByType(t.Context(), fb)
	assert.ErrorIs(t, err, errPairEmpty)
	fb.Pair = testPair
	_, err = bi.GetFeeByType(t.Context(), fb)
	assert.NoError(t, err)
	fb.IsMaker = true
	_, err = bi.GetFeeByType(t.Context(), fb)
	assert.NoError(t, err)
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	err := bi.ValidateAPICredentials(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricCandles(t.Context(), currency.Pair{}, asset.Spot, kline.Raw, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = bi.GetHistoricCandles(t.Context(), testPair, asset.Spot, kline.OneDay, time.Now().Add(-time.Hour*24*20), time.Now())
	assert.NoError(t, err)
	_, err = bi.GetHistoricCandles(t.Context(), testPair, asset.Futures, kline.OneDay, time.Now().Add(-time.Hour*24*20), time.Now())
	assert.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricCandlesExtended(t.Context(), currency.Pair{}, asset.Spot, kline.Raw, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = bi.GetHistoricCandlesExtended(t.Context(), testPair, asset.Spot, kline.OneDay, time.Now().Add(-time.Hour*24*20), time.Now())
	assert.NoError(t, err)
	_, err = bi.GetHistoricCandlesExtended(t.Context(), testPair, asset.Futures, kline.OneDay, time.Now().Add(-time.Hour*24*20), time.Now())
	assert.NoError(t, err)
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetFuturesContractDetails, asset.Empty, asset.Empty, nil, false, false, true)
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	var nilReq *fundingrate.LatestRateRequest
	_, err := bi.GetLatestFundingRates(t.Context(), nilReq)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	req1, req2 := new(fundingrate.LatestRateRequest), new(fundingrate.LatestRateRequest)
	req2.Pair = testPair
	testGetOneArg(t, bi.GetLatestFundingRates, req1, req2, currency.ErrCurrencyPairEmpty, false, false, true)
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	err := bi.UpdateOrderExecutionLimits(t.Context(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = bi.UpdateOrderExecutionLimits(t.Context(), asset.Spot)
	assert.NoError(t, err)
	err = bi.UpdateOrderExecutionLimits(t.Context(), asset.Futures)
	assert.NoError(t, err)
	err = bi.UpdateOrderExecutionLimits(t.Context(), asset.Margin)
	assert.NoError(t, err)
}

func TestUpdateCurrencyStates(t *testing.T) {
	t.Parallel()
	err := bi.UpdateCurrencyStates(t.Context(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetAvailableTransferChains, currency.EMPTYCODE, testCrypto, errCurrencyEmpty, false, false, true)
	_, err := bi.GetAvailableTransferChains(t.Context(), fakeCurrency)
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
	testGetOneArg(t, bi.GetMarginRatesHistory, nil, req2, common.ErrNilPointer, false, true, true)
	req2.Asset = asset.CrossMargin
	testGetOneArg(t, bi.GetMarginRatesHistory, req1, req2, common.ErrDateUnset, false, true, true)
	req1.Asset = asset.CrossMargin
	testGetOneArg(t, bi.GetMarginRatesHistory, req1, nil, common.ErrDateUnset, false, true, false)
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesPositionSummary(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Pair: testPair})
	assert.NoError(t, err)
}

func TestGetFuturesPositions(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesPositions(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	req := &futures.PositionsRequest{
		Pairs: currency.Pairs{testPair, currency.NewPair(currency.BTC, currency.ETH)},
	}
	_, err = bi.GetFuturesPositions(t.Context(), req)
	assert.NoError(t, err)
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	req := new(futures.PositionsRequest)
	testGetOneArg(t, bi.GetFuturesPositionOrders, nil, req, common.ErrNilPointer, false, true, true)
	req.Pairs = currency.Pairs{testPair}
	testGetOneArg(t, bi.GetFuturesPositionOrders, nil, req, nil, false, true, true)
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	req := &fundingrate.HistoricalRatesRequest{
		Pair: testPair,
	}
	testGetOneArg(t, bi.GetHistoricalFundingRates, nil, req, common.ErrNilPointer, false, false, false)
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	err := bi.SetCollateralMode(t.Context(), 0, 0)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCollateralMode, 0, 0, common.ErrFunctionNotSupported, false, true, false)
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	err := bi.SetMarginType(t.Context(), 0, currency.Pair{}, 0)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = bi.SetMarginType(t.Context(), asset.Futures, currency.Pair{}, margin.Isolated)
	assert.ErrorIs(t, err, errPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	err = bi.SetMarginType(t.Context(), asset.Futures, testPair, margin.Multi)
	assert.NoError(t, err)
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.ChangePositionMargin, nil, nil, common.ErrFunctionNotSupported, false, true, false)
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	err := bi.SetLeverage(t.Context(), 0, currency.Pair{}, 0, 0, 0)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	err = bi.SetLeverage(t.Context(), asset.Futures, currency.Pair{}, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	err = bi.SetLeverage(t.Context(), asset.Futures, testPair, 0, 1, order.Long)
	assert.NoError(t, err)
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := bi.GetLeverage(t.Context(), 0, currency.Pair{}, 0, 0)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	_, err = bi.GetLeverage(t.Context(), asset.Futures, currency.Pair{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetLeverage(t.Context(), asset.Futures, testPair, 0, 0)
	assert.ErrorIs(t, err, margin.ErrMarginTypeUnsupported)
	_, err = bi.GetLeverage(t.Context(), asset.Futures, testPair, margin.Isolated, 0)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid)
	_, err = bi.GetLeverage(t.Context(), asset.Futures, testPair, margin.Isolated, order.Long)
	assert.NoError(t, err)
	_, err = bi.GetLeverage(t.Context(), asset.Futures, testPair, margin.Isolated, order.Short)
	assert.NoError(t, err)
	_, err = bi.GetLeverage(t.Context(), asset.Futures, testPair, margin.Multi, 0)
	assert.NoError(t, err)
	_, err = bi.GetLeverage(t.Context(), asset.Margin, currency.Pair{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetLeverage(t.Context(), asset.Margin, testPair, 0, 0)
	assert.NoError(t, err)
	_, err = bi.GetLeverage(t.Context(), asset.CrossMargin, currency.Pair{}, 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.GetLeverage(t.Context(), asset.CrossMargin, testPair, 0, 0)
	assert.NoError(t, err)
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := bi.GetOpenInterest(t.Context(), key.PairAsset{Base: testCrypto.Item, Quote: testFiat.Item, Asset: asset.Futures})
	assert.NoError(t, err)
}

func TestCalculateUpdateOrderbookChecksum(t *testing.T) {
	t.Parallel()
	ord := orderbook.Base{
		Asks: orderbook.Tranches{
			{
				StrPrice:  "3",
				StrAmount: "1",
			},
		},
		Bids: orderbook.Tranches{
			{
				StrPrice:  "4",
				StrAmount: "1",
			},
		},
	}
	err := bi.CalculateUpdateOrderbookChecksum(&ord, 0)
	assert.ErrorIs(t, err, errInvalidChecksum)
	err = bi.CalculateUpdateOrderbookChecksum(&ord, 892106381)
	assert.NoError(t, err)
	ord.Asks = make(orderbook.Tranches, 26)
	data := "3141592653589793238462643383279502884197169399375105"
	for i := range ord.Asks {
		ord.Asks[i] = orderbook.Tranche{
			StrPrice:  string(data[i*2]),
			StrAmount: string(data[i*2+1]),
		}
	}
	err = bi.CalculateUpdateOrderbookChecksum(&ord, 2945115267)
	assert.NoError(t, err)
}

// The following 20 tests aren't parallel due to collisions with each other, and some other tests
func TestCommitConversion(t *testing.T) {
	_, err := bi.CommitConversion(t.Context(), currency.Code{}, currency.Code{}, "", 0, 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.CommitConversion(t.Context(), testCrypto, testFiat, "", 0, 0, 0)
	assert.ErrorIs(t, err, errTraceIDEmpty)
	_, err = bi.CommitConversion(t.Context(), testCrypto, testFiat, "1", 0, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.CommitConversion(t.Context(), testCrypto, testFiat, "1", 1, 1, 0)
	assert.ErrorIs(t, err, errPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetQuotedPrice(t.Context(), testCrypto, testFiat, testAmount, 0)
	require.NoError(t, err)
	_, err = bi.CommitConversion(t.Context(), testCrypto, testFiat, resp.TraceID, resp.FromCoinSize, resp.ToCoinSize, resp.ConvertPrice)
	assert.NoError(t, err)
}

func TestModifyPlanSpotOrder(t *testing.T) {
	_, err := bi.ModifyPlanSpotOrder(t.Context(), 0, "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.ModifyPlanSpotOrder(t.Context(), 0, "meow", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.ModifyPlanSpotOrder(t.Context(), 0, "meow", "woof", 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.ModifyPlanSpotOrder(t.Context(), 0, "meow", "limit", 1, 0, 0)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = bi.ModifyPlanSpotOrder(t.Context(), 0, "meow", "woof", 1, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordID, err := bi.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	if len(ordID.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := bi.ModifyPlanSpotOrder(t.Context(), ordID.OrderList[0].OrderID, ordID.OrderList[0].ClientOrderID, "limit", testPrice, testPrice, testAmount)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestCancelPlanSpotOrder(t *testing.T) {
	_, err := bi.CancelPlanSpotOrder(t.Context(), 0, "")
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordID, err := bi.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	require.NotNil(t, ordID)
	if len(ordID.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := bi.CancelPlanSpotOrder(t.Context(), ordID.OrderList[0].OrderID, ordID.OrderList[0].ClientOrderID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyOrder(t *testing.T) {
	var ord *order.Modify
	_, err := bi.ModifyOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrModifyOrderIsNil)
	ord = &order.Modify{
		Pair:      testPair,
		AssetType: 1<<31 - 1,
		OrderID:   "meow",
	}
	_, err = bi.ModifyOrder(t.Context(), ord)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	ord.OrderID = "0"
	_, err = bi.ModifyOrder(t.Context(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ord.AssetType = asset.Futures
	_, err = bi.ModifyOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordID, err := bi.GetCurrentSpotPlanOrders(t.Context(), testPair, time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	if len(ordID.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	ord.OrderID = strconv.FormatInt(ordID.OrderList[0].OrderID, 10)
	ord.ClientOrderID = ordID.OrderList[0].ClientOrderID
	ord.Type = order.Limit
	ord.Price = testPrice
	ord.TriggerPrice = testPrice
	ord.Amount = testAmount
	ord.AssetType = asset.Spot
	_, err = bi.ModifyOrder(t.Context(), ord)
	assert.NoError(t, err)
}

func TestCancelTriggerFuturesOrders(t *testing.T) {
	var ordList []OrderIDStruct
	_, err := bi.CancelTriggerFuturesOrders(t.Context(), ordList, currency.Pair{}, "", "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordList = append(ordList, OrderIDStruct{
		OrderID:       1,
		ClientOrderID: "a",
	})
	resp, err := bi.CancelTriggerFuturesOrders(t.Context(), ordList, testPair2, testFiat2.String()+"-FUTURES", "", currency.Code{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestRepayLoan(t *testing.T) {
	_, err := bi.RepayLoan(t.Context(), 0, 0, false, false)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = bi.RepayLoan(t.Context(), 1, 0, false, false)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetOngoingLoans(t.Context(), 0, currency.Code{}, currency.Code{})
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.RepayLoan(t.Context(), resp[0].OrderID, testAmount, false, false)
	assert.NoError(t, err)
	_, err = bi.RepayLoan(t.Context(), resp[0].OrderID, 0, true, true)
	assert.NoError(t, err)
}

func TestModifyFuturesOrder(t *testing.T) {
	_, err := bi.ModifyFuturesOrder(t.Context(), 0, "", "", "", currency.Pair{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.ModifyFuturesOrder(t.Context(), 1, "", "", "", currency.Pair{}, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.ModifyFuturesOrder(t.Context(), 1, "", "", "", currency.NewPairWithDelimiter("meow", "woof", ""), 0, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ModifyFuturesOrder(t.Context(), 1, "", "meow", "", currency.NewPairWithDelimiter("meow", "woof", ""), 0, 0, 0, 0)
	assert.ErrorIs(t, err, errNewClientOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.ModifyFuturesOrder(t.Context(), 1, "a", testFiat2.String()+"-FUTURES", clientIDGenerator(), testPair2, testAmount2+1, testPrice2+2, testPrice2+1, testPrice2/10)
	assert.NoError(t, err)
}

func TestCancelFuturesOrder(t *testing.T) {
	_, err := bi.CancelFuturesOrder(t.Context(), currency.Pair{}, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelFuturesOrder(t.Context(), testPair2, "", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.CancelFuturesOrder(t.Context(), testPair2, "woof", "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CancelFuturesOrder(t.Context(), testPair2, testFiat2.String()+"-FUTURES", "", testFiat2, 1)
	assert.NoError(t, err)
}

func TestBatchCancelFuturesOrders(t *testing.T) {
	_, err := bi.BatchCancelFuturesOrders(t.Context(), nil, currency.Pair{}, "", currency.Code{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	orders := []OrderIDStruct{
		{
			OrderID:       1,
			ClientOrderID: "a",
		},
	}
	_, err = bi.BatchCancelFuturesOrders(t.Context(), orders, testPair2, testFiat2.String()+"-FUTURES", testFiat2)
	assert.NoError(t, err)
}

func TestFlashClosePosition(t *testing.T) {
	_, err := bi.FlashClosePosition(t.Context(), currency.Pair{}, "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.FlashClosePosition(t.Context(), testPair2, "", testFiat2.String()+"-FUTURES")
	assert.NoError(t, err)
}

func TestCancelAllFuturesOrders(t *testing.T) {
	_, err := bi.CancelAllFuturesOrders(t.Context(), currency.Pair{}, "", currency.Code{}, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CancelAllFuturesOrders(t.Context(), currency.Pair{}, testFiat2.String()+"-FUTURES", testFiat2, time.Second*60)
	assert.NoError(t, err)
}

func TestCancelOrder(t *testing.T) {
	var ord *order.Cancel
	err := bi.CancelOrder(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil)
	ord = &order.Cancel{
		OrderID:   "meow",
		AssetType: 1<<31 - 1,
	}
	err = bi.CancelOrder(t.Context(), ord)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	ord.OrderID = "0"
	err = bi.CancelOrder(t.Context(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ord.AssetType = asset.Margin
	err = bi.CancelOrder(t.Context(), ord)
	assert.ErrorIs(t, err, errPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetUnfilledOrders(t.Context(), testPair, "", time.Time{}, time.Time{}, 5, 1<<62, 0, time.Minute)
	require.NoError(t, err)
	if len(resp) != 0 {
		ord.OrderID = strconv.FormatInt(int64(resp[0].OrderID), 10)
		ord.Pair = testPair
		ord.AssetType = asset.Spot
		ord.ClientOrderID = resp[0].ClientOrderID
		err = bi.CancelOrder(t.Context(), ord)
		assert.NoError(t, err)
	}
	ord.OrderID = "1"
	ord.Pair = testPair2
	ord.AssetType = asset.Futures
	ord.ClientOrderID = "a"
	err = bi.CancelOrder(t.Context(), ord)
	assert.NoError(t, err)
	ord.OrderID = "2"
	ord.Pair = testPair
	ord.AssetType = asset.CrossMargin
	ord.ClientOrderID = "b"
	err = bi.CancelOrder(t.Context(), ord)
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	var ord *order.Cancel
	_, err := bi.CancelAllOrders(t.Context(), ord)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil)
	ord = &order.Cancel{
		AssetType: asset.Empty,
	}
	_, err = bi.CancelAllOrders(t.Context(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ord.AssetType = asset.Spot
	ord.Pair = testPair
	_, err = bi.CancelAllOrders(t.Context(), ord)
	assert.NoError(t, err)
	ord.AssetType = asset.Futures
	ord.Pair = testPair2
	_, err = bi.CancelAllOrders(t.Context(), ord)
	assert.NoError(t, err)
}

func TestCancelCrossOrder(t *testing.T) {
	_, err := bi.CancelCrossOrder(t.Context(), currency.Pair{}, "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelCrossOrder(t.Context(), testPair, "", 0)
	assert.ErrorIs(t, err, errOrderIDMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CancelCrossOrder(t.Context(), testPair, "", 1)
	assert.NoError(t, err)
	_, err = bi.CancelCrossOrder(t.Context(), testPair, "a", 0)
	assert.NoError(t, err)
}

func TestBatchCancelCrossOrders(t *testing.T) {
	_, err := bi.BatchCancelCrossOrders(t.Context(), currency.Pair{}, nil)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchCancelCrossOrders(t.Context(), testPair, nil)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.BatchCancelCrossOrders(t.Context(), testPair, []OrderIDStruct{{
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
	_, err := bi.CancelBatchOrders(t.Context(), orders)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	orders[0].OrderID = "0"
	_, err = bi.CancelBatchOrders(t.Context(), orders)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	orders = nil
	orders = append(orders, order.Cancel{
		AssetType:     asset.Spot,
		OrderID:       "1",
		ClientOrderID: "a",
		Pair:          testPair,
	})
	orders = append(orders, order.Cancel{
		AssetType:     asset.Futures,
		OrderID:       "2",
		ClientOrderID: "b",
		Pair:          testPair2,
	})
	orders = append(orders, order.Cancel{
		AssetType:     asset.Margin,
		OrderID:       "3",
		ClientOrderID: "c",
		Pair:          testPair,
	})
	orders = append(orders, order.Cancel{
		AssetType:     asset.CrossMargin,
		OrderID:       "4",
		ClientOrderID: "d",
		Pair:          testPair,
	})
	_, err = bi.CancelBatchOrders(t.Context(), orders)
	assert.NoError(t, err)
}

func TestCancelIsolatedOrder(t *testing.T) {
	_, err := bi.CancelIsolatedOrder(t.Context(), currency.Pair{}, "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelIsolatedOrder(t.Context(), testPair2, "", 0)
	assert.ErrorIs(t, err, errOrderIDMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CancelIsolatedOrder(t.Context(), testPair, "", 1)
	assert.NoError(t, err)
	_, err = bi.CancelIsolatedOrder(t.Context(), testPair, "a", 0)
	assert.NoError(t, err)
}

func TestBatchCancelIsolatedOrders(t *testing.T) {
	_, err := bi.BatchCancelIsolatedOrders(t.Context(), currency.Pair{}, nil)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchCancelIsolatedOrders(t.Context(), testPair2, nil)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.BatchCancelIsolatedOrders(t.Context(), testPair, []OrderIDStruct{{
		OrderID:       1,
		ClientOrderID: "a",
	}})
	assert.NoError(t, err)
}

func TestWsAuth(t *testing.T) {
	bi.Websocket.SetCanUseAuthenticatedEndpoints(false)
	err := bi.WsAuth(t.Context(), nil)
	assert.ErrorIs(t, err, errAuthenticatedWebsocketDisabled)
	if bi.Websocket.IsEnabled() && !bi.API.AuthenticatedWebsocketSupport || !sharedtestvalues.AreAPICredentialsSet(bi) {
		t.Skip(stream.ErrWebsocketNotEnabled.Error())
	}
	bi.Websocket.SetCanUseAuthenticatedEndpoints(true)
	var dialer websocket.Dialer
	go func() {
		timer := time.NewTimer(sharedtestvalues.WebsocketResponseDefaultTimeout)
		select {
		case resp := <-bi.Websocket.DataHandler:
			t.Errorf("%+v\n%T\n", resp, resp)
		case <-timer.C:
		}
		timer.Stop()
		for {
			<-bi.Websocket.DataHandler
		}
	}()
	err = bi.WsAuth(t.Context(), &dialer)
	require.NoError(t, err)
	time.Sleep(sharedtestvalues.WebsocketResponseDefaultTimeout)
}

// func TestWsReadData(t *testing.T) {
// 	mock := func(tb testing.TB, msg []byte, w *websocket.Conn) error {
// 		tb.Helper()
// 		return nil
// 	}
// 	wsTest := testexch.MockWsInstance[Bitget](t, mockws.CurryWsMockUpgrader(t, mock))
// 	wsTest.Websocket.Enable()
// 	err := wsTest.Subscribe(defaultSubscriptions)
// 	require.NoError(t, err)
// 	// Implement internal/testing/websocket mockws stuff after merging
// 	// See: https://github.com/thrasher-corp/gocryptotrader/blob/master/exchanges/kraken/kraken_test.go#L1169
// }

func TestWsHandleData(t *testing.T) {
	// Not sure what issues this is preventing. If you figure that out, add a comment about it
	// ch := make(chan struct{})
	// t.Cleanup(func() {
	// 	close(ch)
	// })
	// go func() {
	// 	for {
	// 		select {
	// 		case <-bi.Websocket.DataHandler:
	// 			continue
	// 		case <-ch:
	// 			return
	// 		}
	// 	}
	// }()
	verboseTemp := bi.Verbose
	bi.Verbose = true
	t.Cleanup(func() {
		bi.Verbose = verboseTemp
	})
	mockJSON := []byte(`pong`)
	err := bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`notjson`)
	err = bi.wsHandleData(mockJSON)
	errInvalidChar := "invalid char"
	assert.ErrorContains(t, err, errInvalidChar)
	mockJSON = []byte(`{"event":"subscribe"}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"error"}`)
	err = bi.wsHandleData(mockJSON)
	expectedErr := fmt.Sprintf(errWebsocketGeneric, "Bitget", 0, "")
	assert.EqualError(t, err, expectedErr)
	mockJSON = []byte(`{"event":"login","code":0}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"login","code":1}`)
	err = bi.wsHandleData(mockJSON)
	expectedErr = fmt.Sprintf(errWebsocketLoginFailed, "Bitget", "")
	assert.EqualError(t, err, expectedErr)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fakeChannelNotReal"}}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"fakeChannelNotReal"}}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"fakeEventNotReal"}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestTickerDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"SPOT"},"data":[[]]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"SPOT"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"SPOT"},"data":[{"InstId":"BTCUSD"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"USDT-FUTURES"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"USDT-FUTURES"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"USDT-FUTURES"},"data":[{"InstId":"BTCUSD"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"ticker","instType":"moo"},"data":[{"InstId":"BTCUSD"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestCandleDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"candle1D"},"data":[["1","2","3","4","5","6","",""]]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["a","2","3","4","5","6","",""]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","a","3","4","5","6","",""]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","a","4","5","6","",""]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","a","5","6","",""]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","4","a","6","",""]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","4","5","a","",""]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"candle1D","instId":"BTCUSD"},"data":[["1","2","3","4","5","6","",""]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"candle1D"},"data":[[[{}]]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
}

func TestTradeDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"trade","instId":"BTCUSD"},"data":[[]]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"trade","instId":"BTCUSD"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"trade"},"data":[]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
}

func TestOrderbookDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"books"},"data":[]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errReturnEmpty)
	mockJSON = []byte(`{"action":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[{"bids":[["a","1"]]}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"action":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[{"asks":[["1","a"]]}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	mockJSON = []byte(`{"action":"snapshot","arg":{"channel":"books","instId":"BTCUSD"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, orderbook.ErrAssetTypeNotSet)
	mockJSON = []byte(`{"action":"update","arg":{"channel":"books","instId":"BTCUSD"},"data":[{"asks":[["1","2"]]}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, buffer.ErrDepthNotFound)
	mockJSON = []byte(`{"action":"update","arg":{"channel":"books","instId":"BTCUSD"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestAccountSnapshotDataHandler(t *testing.T) {
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"account"},"data":[]}`)
	err := bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"spot"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"spot"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"futures"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account","instType":"futures"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestFillDataHandler(t *testing.T) {
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"fill"},"data":[]}`)
	err := bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"spot"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"spot"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"spot"},"data":[{"symbol":"BTCUSD"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"futures"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"futures"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"fill","instType":"futures"},"data":[{"symbol":"BTCUSD"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestGenOrderDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"orders"},"data":[]}`)
	err := bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[{"instId":"BTCUSD","side":"buy","orderType":"limit","feeDetail":[{}]}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"spot"},"data":[{"instId":"BTCUSD","side":"sell"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"futures"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"futures"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"futures"},"data":[{"instId":"BTCUSD","side":"buy","orderType":"limit","feeDetail":[{}]}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders","instType":"futures"},"data":[{"instId":"BTCUSD","side":"sell"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestTriggerOrderDatHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"orders-algo"},"data":[]}`)
	err := bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"spot"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"spot"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"spot"},"data":[{"instId":"BTCUSD"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"futures"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"futures"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-algo","instType":"futures"},"data":[{"instId":"BTCUSD"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestPositionsDataHandler(t *testing.T) {
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"positions"},"data":[[]]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions"},"data":[{"instId":"BTCUSD"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestPositionsHistoryDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"positions-history"},"data":[[]]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions-history"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"positions-history"},"data":[{"instId":"BTCUSD"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestIndexPriceDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"index-price"},"data":[[]]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"index-price","instType":"spot"},"data":[{"symbol":"BTCUSDT"},{"symbol":"USDT/USDT"}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestCrossAccountDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"account-crossed"},"data":[[]]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account-crossed"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestMarginOrderDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"orders-crossed"},"data":[[]]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-crossed"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorIs(t, err, errUnknownPairQuote)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-isolated","instId":"BTCUSD"},"data":[{"feeDetail":[{}]}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"orders-crossed","instId":"BTCUSD"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestIsolatedAccountDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"snapshot","arg":{"channel":"account-isolated"},"data":[[]]}`)
	err := bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"snapshot","arg":{"channel":"account-isolated"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
}

func TestAccountUpdateDataHandler(t *testing.T) {
	t.Parallel()
	mockJSON := []byte(`{"event":"update","arg":{"channel":"account"},"data":[]}`)
	err := bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"spot"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"spot"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"futures"},"data":[[]]}`)
	err = bi.wsHandleData(mockJSON)
	assert.ErrorContains(t, err, errUnmarshalArray)
	mockJSON = []byte(`{"event":"update","arg":{"channel":"account","instType":"futures"},"data":[{}]}`)
	err = bi.wsHandleData(mockJSON)
	assert.NoError(t, err)
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
	string | int64 | []string | bool | asset.Item | *fundingrate.LatestRateRequest | currency.Code | *margin.RateHistoryRequest | *fundingrate.HistoricalRatesRequest | *futures.PositionsRequest | *withdraw.Request | *margin.PositionChangeRequest | currency.Pair | []currency.Code | currency.Pairs
}

type getOneArgGen[R getOneArgResp, P getOneArgParam] func(context.Context, P) (R, error)

func testGetOneArg[R getOneArgResp, P getOneArgParam](t *testing.T, f getOneArgGen[R, P], callErrCheck, callNoEErr P, tarErr error, checkResp, checkCreds, canManipOrders bool) {
	t.Helper()
	if tarErr != nil {
		_, err := f(t.Context(), callErrCheck)
		assert.ErrorIs(t, err, tarErr)
	}
	if checkCreds {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipOrders)
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
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = f(t.Context(), currency.NewPairWithDelimiter("meow", "woof", ""), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = f(t.Context(), testPair2, testFiat2.String()+"-FUTURES")
	assert.NoError(t, err)
}

func subAccTestHelper(t *testing.T, compString, ignoreString string) string {
	t.Helper()
	resp, err := bi.GetVirtualSubaccounts(t.Context(), 25, 0, "")
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

func exchangeBaseHelper(bi *Bitget) error {
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
	bi.Websocket = sharedtestvalues.NewTestWebsocket()
	err = bi.Setup(exchCfg)
	if err != nil {
		return err
	}
	return nil
}

func riskUnitHelper(t *testing.T) string {
	t.Helper()
	resp, err := bi.GetRiskUnit(t.Context())
	require.NoError(t, err)
	if len(resp) == 0 {
		t.Skip(skipInsufficientRiskUnits)
	}
	return resp[0]
}
