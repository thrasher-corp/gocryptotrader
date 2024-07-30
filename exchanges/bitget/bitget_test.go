package bitget

import (
	"context"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
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

	// Needs to be set to a subaccount with deposit transfer permissions so that TestGetSubaccountDepositAddress
	// doesn't fail
	deposSubaccID = ""

	testSubaccountName = "GCTTESTA"
	testIP             = "14.203.57.50"
	testAddress        = "fake test address"
	// Test values used with live data, with the goal of never letting an order be executed
	testAmount = 0.001
	testPrice  = 1e10 - 1
	// Test values used with demo functionality, with the goal of lining up with the relatively strict currency
	// limits present there
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
	ordersNotFound = "Orders not found"

	skipTestSubAccNotFound       = "appropriate sub-account (equals %v, not equals %v) not found, skipping"
	skipInsufficientAPIKeysFound = "insufficient API keys found, skipping"
	skipInsufficientBalance      = "insufficient balance to place order, skipping"
	skipInsufficientOrders       = "insufficient orders found, skipping"

	errAPIKeyLimitPartial              = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"40063","msg":"API exceeds the maximum limit added","requestTime":`
	errCurrentlyHoldingPositionPartial = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"45117","msg":"Currently holding positions or orders, the margin mode cannot be adjusted","requestTime":`
	errFakePairDoesNotExistPartial     = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"40034","msg":"Parameter FAKEPAIRNOTREALMEOWMEOW does not exist","requestTime"`
)

// Developer-defined variables to aid testing
var (
	fakePair = currency.NewPair(currency.NewCode("FAKEPAIRNOT"), currency.NewCode("REALMEOWMEOW"))
)

var bi = &Bitget{}

func TestMain(m *testing.M) {
	bi.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bitget")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.ClientID = clientID

	err = bi.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	bi.Verbose = true

	os.Exit(m.Run())
}

func TestInterface(t *testing.T) {
	t.Parallel()
	var e exchange.IBotExchange
	e = new(Bitget)
	_, ok := e.(exchange.IBotExchange)
	assert.True(t, ok)
}

func TestQueryAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := bi.QueryAnnouncements(context.Background(), "", time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := bi.QueryAnnouncements(context.Background(), "latest_news", time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetTime(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetTime)
}

func TestGetTradeRate(t *testing.T) {
	t.Parallel()
	_, err := bi.GetTradeRate(context.Background(), "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetTradeRate(context.Background(), testPair.String(), "")
	assert.ErrorIs(t, err, errBusinessTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetTradeRate(context.Background(), testPair.String(), "spot")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSpotTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotTransactionRecords(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotTransactionRecords(context.Background(), "", time.Now().Add(-time.Hour*24*30), time.Now(), 5,
		1<<62)
	assert.NoError(t, err)
}

func TestGetFuturesTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesTransactionRecords(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesTransactionRecords(context.Background(), "woof", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesTransactionRecords(context.Background(), "COIN-FUTURES", "",
		time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetMarginTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMarginTransactionRecords(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetMarginTransactionRecords(context.Background(), "", "", time.Now().Add(-time.Hour*24*30),
		time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetP2PTransactionRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetP2PTransactionRecords(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetP2PTransactionRecords(context.Background(), "", time.Now().Add(-time.Hour*24*30), time.Now(), 5,
		1<<62)
	assert.NoError(t, err)
}

func TestGetP2PMerchantList(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetP2PMerchantList(context.Background(), "", 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetMerchantInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetMerchantInfo)
}

func TestGetMerchantP2POrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMerchantP2POrders(context.Background(), time.Time{}, time.Time{}, 0, 0, 0, 0, "", "", "", "")
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// Can't currently be properly tested due to not knowing any p2p order IDs
	_, err = bi.GetMerchantP2POrders(context.Background(), time.Now().Add(-time.Hour*24*7), time.Now(), 5, 0, 0,
		0, "", "", "", "")
	assert.NoError(t, err)
}

func TestGetMerchantAdvertisementList(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMerchantAdvertisementList(context.Background(), time.Time{}, time.Time{}, 0, 0, 0, 0, "", "", "",
		"", "", "")
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetMerchantAdvertisementList(context.Background(), time.Now().Add(-time.Hour*24*7), time.Now(), 5,
		1<<62, 0, 0, "", "", "", "", "", "")
	assert.NoError(t, err)
}

func TestGetSpotWhaleNetFlow(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetSpotWhaleNetFlow, "", testPair.String(), errPairEmpty, true, false, false)
}

func TestGetFuturesActiveVolume(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesActiveVolume(context.Background(), "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetFuturesActiveVolume(context.Background(), testPair.String(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetFuturesPositionRatios(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesPositionRatios(context.Background(), "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetFuturesPositionRatios(context.Background(), testPair.String(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

// Test for an endpoint which doesn't work
// func TestGetMarginPositionRatios(t *testing.T) {
// 	t.Parallel()
// 	_, err := bi.GetMarginPositionRatios(context.Background(), "", "", "")
// 	assert.ErrorIs(t, err, errPairEmpty)
// 	resp, err := bi.GetMarginPositionRatios(context.Background(), testPair.String(), "", "")
// 	require.NoError(t, err)
// 	assert.NotEmpty(t, resp.Data)
// }

// Test for an endpoint which doesn't work
// func TestGetMarginLoanGrowth(t *testing.T) {
// 	t.Parallel()
// 	_, err := bi.GetMarginLoanGrowth(context.Background(), "", "", "")
// 	assert.ErrorIs(t, err, errPairEmpty)
// 	resp, err := bi.GetMarginLoanGrowth(context.Background(), testPair.String(), "", "")
// 	require.NoError(t, err)
// 	assert.NotEmpty(t, resp.Data)
// }

// Test for an endpoint which doesn't work
// func TestGetIsolatedBorrowingRatio(t *testing.T) {
// 	t.Parallel()
// 	_, err := bi.GetIsolatedBorrowingRatio(context.Background(), "", "")
// 	assert.ErrorIs(t, err, errPairEmpty)
// 	resp, err := bi.GetIsolatedBorrowingRatio(context.Background(), testPair.String(), "")
// 	require.NoError(t, err)
// 	assert.NotEmpty(t, resp.Data)
// }

func TestGetFuturesRatios(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesRatios(context.Background(), "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetFuturesRatios(context.Background(), testPair.String(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSpotFundFlows(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetSpotFundFlows, "", testPair.String(), errPairEmpty, true, false, false)
}

func TestGetTradeSupportSymbols(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetTradeSupportSymbols)
}

func TestGetSpotWhaleFundFlows(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetSpotWhaleFundFlows, "", testPair.String(), errPairEmpty, true, false, false)
}

func TestGetFuturesAccountRatios(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesAccountRatios(context.Background(), "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetFuturesAccountRatios(context.Background(), testPair.String(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestCreateVirtualSubaccounts(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.CreateVirtualSubaccounts, nil, []string{testSubaccountName}, errSubaccountEmpty,
		true, true, true)
}

func TestModifyVirtualSubaccount(t *testing.T) {
	t.Parallel()
	perms := []string{}
	_, err := bi.ModifyVirtualSubaccount(context.Background(), "", "", perms)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.ModifyVirtualSubaccount(context.Background(), "meow", "", perms)
	assert.ErrorIs(t, err, errNewStatusEmpty)
	_, err = bi.ModifyVirtualSubaccount(context.Background(), "meow", "woof", perms)
	assert.ErrorIs(t, err, errNewPermsEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	tarID := subAccTestHelper(t, strings.ToLower(string(testSubaccountName[:3]))+"****@virtual-bitget.com", "")
	perms = append(perms, "read")
	resp, err := bi.ModifyVirtualSubaccount(context.Background(), tarID, "normal", perms)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestCreateSubaccountAndAPIKey(t *testing.T) {
	t.Parallel()
	ipL := []string{}
	_, err := bi.CreateSubaccountAndAPIKey(context.Background(), "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ipL = append(ipL, testIP)
	pL := []string{"read"}
	// Fails with error "subAccountList not empty" and I'm not sure why. The account I'm testing with is far off
	// hitting the limit of 20 sub-accounts.
	// Now it's saying that parameter req cannot be empty, still no clue what that means
	_, err = bi.CreateSubaccountAndAPIKey(context.Background(), "MEOWMEOW", "woofwoof", "neighneighneighneighneigh",
		ipL, pL)
	assert.NoError(t, err)
}

func TestGetVirtualSubaccounts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetVirtualSubaccounts(context.Background(), 25, 1, "")
	assert.NoError(t, err)
}

func TestCreateAPIKey(t *testing.T) {
	t.Parallel()
	ipL := []string{}
	_, err := bi.CreateAPIKey(context.Background(), "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.CreateAPIKey(context.Background(), "woof", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errPassphraseEmpty)
	_, err = bi.CreateAPIKey(context.Background(), "woof", "meow", "", ipL, ipL)
	assert.ErrorIs(t, err, errLabelEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	tarID := subAccTestHelper(t, strings.ToLower(string(testSubaccountName[:3]))+"****@virtual-bitget.com", "")
	ipL = append(ipL, testIP)
	pL := []string{"read"}
	_, err = bi.CreateAPIKey(context.Background(), tarID, clientID, "neigh whinny", ipL, pL)
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
	_, err := bi.ModifyAPIKey(context.Background(), "", "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errAPIKeyEmpty)
	_, err = bi.ModifyAPIKey(context.Background(), "", "", "", "woof", ipL, ipL)
	assert.ErrorIs(t, err, errPassphraseEmpty)
	_, err = bi.ModifyAPIKey(context.Background(), "", "meow", "", "woof", ipL, ipL)
	assert.ErrorIs(t, err, errLabelEmpty)
	_, err = bi.ModifyAPIKey(context.Background(), "", "meow", "quack", "woof", ipL, ipL)
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	tarID := subAccTestHelper(t, strings.ToLower(string(testSubaccountName[:3]))+"****@virtual-bitget.com", "")
	resp, err := bi.GetAPIKeys(context.Background(), tarID)
	assert.NoError(t, err)
	if len(resp.Data) == 0 {
		t.Skip(skipInsufficientAPIKeysFound)
	}
	resp2, err := bi.ModifyAPIKey(context.Background(), tarID, clientID, "oink", resp.Data[0].SubaccountApiKey,
		ipL, ipL)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2.Data)
}

func TestGetAPIKeys(t *testing.T) {
	t.Parallel()
	var tarID string
	if sharedtestvalues.AreAPICredentialsSet(bi) {
		tarID = subAccTestHelper(t, strings.ToLower(string(testSubaccountName[:3]))+"****@virtual-bitget.com", "")
	}
	testGetOneArg(t, bi.GetAPIKeys, "", tarID, errSubaccountEmpty, true, true, true)
}

func TestGetFundingAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetFundingAssets, "", testCrypto.String(), nil, false, true, true)
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
	_, err := bi.GetQuotedPrice(context.Background(), "", "", 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.GetQuotedPrice(context.Background(), "meow", "woof", 0, 0)
	assert.ErrorIs(t, err, errFromToMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), 0, 1)
	assert.NoError(t, err)
	resp, err := bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), 0.1, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetConvertHistory(context.Background(), time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetConvertHistory(context.Background(), time.Now().Add(-time.Hour*90*24), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetBGBConvertCoins(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetBGBConvertCoins)
}

func TestConvertBGB(t *testing.T) {
	t.Parallel()
	// No matter what currency I use, this returns the error "currency does not support convert"; possibly a bad
	// error message, with the true issue being lack of funds?
	testGetOneArg(t, bi.ConvertBGB, nil, []string{testCrypto3.String()}, errCurrencyEmpty, false, true,
		canManipulateRealOrders)
}

func TestGetBGBConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetBGBConvertHistory(context.Background(), 0, 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetBGBConvertHistory(context.Background(), 0, 5, 1<<62, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetCoinInfo(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCoinInfo, "", "", nil, true, false, false)
}

func TestGetSymbolInfo(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetSymbolInfo, "", "", nil, true, false, false)
}

func TestGetSpotVIPFeeRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetSpotVIPFeeRate)
}

func TestGetSpotTickerInformation(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetSpotTickerInformation, "", testPair.String(), nil, true, false, false)
}

func TestGetSpotMergeDepth(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotMergeDepth(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetSpotMergeDepth(context.Background(), testPair.String(), "scale3", "5")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetOrderbookDepth(t *testing.T) {
	t.Parallel()
	resp, err := bi.GetOrderbookDepth(context.Background(), testPair.String(), "", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSpotCandlestickData(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotCandlestickData(context.Background(), "", "", time.Time{}, time.Time{}, 0, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotCandlestickData(context.Background(), "meow", "", time.Time{}, time.Time{}, 0, false)
	assert.ErrorIs(t, err, errGranEmpty)
	_, err = bi.GetSpotCandlestickData(context.Background(), "meow", "woof", time.Time{}, time.Time{}, 5, true)
	assert.ErrorIs(t, err, errEndTimeEmpty)
	_, err = bi.GetSpotCandlestickData(context.Background(), "meow", "woof", time.Now().Add(time.Hour), time.Time{},
		0, false)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	_, err = bi.GetSpotCandlestickData(context.Background(), testPair.String(), "1min", time.Time{}, time.Time{},
		5, false)
	assert.NoError(t, err)
	resp, err := bi.GetSpotCandlestickData(context.Background(), testPair.String(), "1min", time.Time{}, time.Now(), 5,
		true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.SpotCandles)
}

func TestGetRecentSpotFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetRecentSpotFills(context.Background(), "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetRecentSpotFills(context.Background(), testPair.String(), 5)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSpotMarketTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotMarketTrades(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotMarketTrades(context.Background(), "meow", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := bi.GetSpotMarketTrades(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestPlaceSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceSpotOrder(context.Background(), "", "", "", "", "", 0, 0, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceSpotOrder(context.Background(), testPair.String(), "", "", "", "", 0, 0, false)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceSpotOrder(context.Background(), testPair.String(), "sell", "", "", "", 0, 0, false)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceSpotOrder(context.Background(), testPair.String(), "sell", "limit", "", "", 0, 0, false)
	assert.ErrorIs(t, err, errStrategyEmpty)
	_, err = bi.PlaceSpotOrder(context.Background(), testPair.String(), "sell", "limit", "IOC", "", 0, 0, false)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = bi.PlaceSpotOrder(context.Background(), testPair.String(), "sell", "limit", "IOC", "", testPrice, 0,
		false)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceSpotOrder(context.Background(), testPair.String(), "sell", "limit", "IOC", "", testPrice,
		testAmount, true)
	assert.NoError(t, err)
}

func TestCancelSpotOrderByID(t *testing.T) {
	t.Parallel()
	_, err := bi.CancelSpotOrderByID(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelSpotOrderByID(context.Background(), testPair.String(), "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetUnfilledOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62, 0)
	require.NoError(t, err)
	if len(resp.Data) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.CancelSpotOrderByID(context.Background(), testPair.String(), resp.Data[0].ClientOrderID,
		int64(resp.Data[0].OrderID))
	assert.NoError(t, err)
}

func TestBatchPlaceSpotOrders(t *testing.T) {
	t.Parallel()
	var req []PlaceSpotOrderStruct
	_, err := bi.BatchPlaceSpotOrders(context.Background(), "", req, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchPlaceSpotOrders(context.Background(), "meow", req, false)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	req = append(req, PlaceSpotOrderStruct{
		Side:      "sell",
		OrderType: "limit",
		Strategy:  "IOC",
		Price:     testPrice,
		Size:      testAmount,
	})
	resp, err := bi.BatchPlaceSpotOrders(context.Background(), testPair.String(), req, true)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestBatchCancelOrders(t *testing.T) {
	t.Parallel()
	var req []OrderIDStruct
	_, err := bi.BatchCancelOrders(context.Background(), "", req)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchCancelOrders(context.Background(), "meow", req)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetUnfilledOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62, 0)
	require.NoError(t, err)
	if len(resp.Data) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	req = append(req, OrderIDStruct{
		OrderID:       int64(resp.Data[0].OrderID),
		ClientOrderID: resp.Data[0].ClientOrderID,
	})
	resp2, err := bi.BatchCancelOrders(context.Background(), testPair.String(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2.Data)
}

func TestCancelOrderBySymbol(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.CancelOrderBySymbol, "", testPair.String(), errPairEmpty, true, true,
		canManipulateRealOrders)
}

func TestGetSpotOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotOrderDetails(context.Background(), 0, "")
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	ordIDs := getPlanOrdIDHelper(t, false)
	_, err = bi.GetSpotOrderDetails(context.Background(), ordIDs.OrderID, ordIDs.ClientOrderID)
	assert.NoError(t, err)
}

func TestGetUnfilledOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetUnfilledOrders(context.Background(), "", time.Now().Add(time.Hour), time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetUnfilledOrders(context.Background(), "", time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
}

func TestGetHistoricalSpotOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalSpotOrders(context.Background(), "", time.Now().Add(time.Hour), time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetHistoricalSpotOrders(context.Background(), "", time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
}

func TestGetSpotFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotFills(context.Background(), "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotFills(context.Background(), "meow", time.Now().Add(time.Hour), time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotFills(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
}

func TestPlacePlanSpotOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlacePlanSpotOrder(context.Background(), "", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlacePlanSpotOrder(context.Background(), "meow", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlacePlanSpotOrder(context.Background(), "meow", "woof", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.PlacePlanSpotOrder(context.Background(), "meow", "woof", "", "", "", "", "", 1, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlacePlanSpotOrder(context.Background(), "meow", "woof", "limit", "", "", "", "", 1, 0, 0)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = bi.PlacePlanSpotOrder(context.Background(), "meow", "woof", "neigh", "", "", "", "", 1, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.PlacePlanSpotOrder(context.Background(), "meow", "woof", "neigh", "", "", "", "", 1, 0, 1)
	assert.ErrorIs(t, err, errTriggerTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.PlacePlanSpotOrder(context.Background(), testPair.String(), "sell", "limit", "", "fill_price",
		clientIDGenerator(), "ioc", testPrice, testPrice, testAmount)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetCurrentSpotPlanOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCurrentSpotPlanOrders(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCurrentSpotPlanOrders(context.Background(), "meow", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetCurrentSpotPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestSpotGetPlanSubOrder(t *testing.T) {
	t.Parallel()
	var ordIDs *OrderIDStruct
	if sharedtestvalues.AreAPICredentialsSet(bi) {
		ordIDs = getPlanOrdIDHelper(t, true)
	}
	// This gets the error "the current plan order does not exist or has not been triggered" even when using
	// a plan order that definitely exists and has definitely been triggered. Re-investigate later
	testGetOneArg(t, bi.GetSpotPlanSubOrder, "", strconv.FormatInt(ordIDs.OrderID, 10), errOrderIDEmpty, true,
		true, true)
}

func TestGetSpotPlanOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotPlanOrderHistory(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotPlanOrderHistory(context.Background(), "meow", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotPlanOrderHistory(context.Background(), testPair.String(), time.Now().Add(-time.Hour*24*90),
		time.Now().Add(-time.Minute), 2, 1<<62)
	assert.NoError(t, err)
}

func TestBatchCancelSpotPlanOrders(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.BatchCancelSpotPlanOrders, nil, []string{testPair.String()}, nil, false, true,
		canManipulateRealOrders)
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// Not chucked into testGetNoArgs due to checking the presence of resp.Data, refactoring that generic for that
	// would waste too many lines to do so just for this
	resp, err := bi.GetAccountInfo(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetAccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetAccountAssets(context.Background(), "", "")
	assert.NoError(t, err)
}

func TestGetSpotSubaccountAssets(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetSpotSubaccountAssets)
}

func TestModifyDepositAccount(t *testing.T) {
	t.Parallel()
	_, err := bi.ModifyDepositAccount(context.Background(), "", "")
	assert.ErrorIs(t, err, errAccountTypeEmpty)
	_, err = bi.ModifyDepositAccount(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ModifyDepositAccount(context.Background(), "spot", testFiat.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Success)
}

func TestGetAccountBills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotAccountBills(context.Background(), "", "", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotAccountBills(context.Background(), testCrypto.String(), "", "", time.Time{}, time.Time{}, 3, 1<<62)
	assert.NoError(t, err)
}

func TestTransferAsset(t *testing.T) {
	t.Parallel()
	_, err := bi.TransferAsset(context.Background(), "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.TransferAsset(context.Background(), "meow", "", "", "", "", 0)
	assert.ErrorIs(t, err, errToTypeEmpty)
	_, err = bi.TransferAsset(context.Background(), "meow", "woof", "", "", "", 0)
	assert.ErrorIs(t, err, errCurrencyAndPairEmpty)
	_, err = bi.TransferAsset(context.Background(), "meow", "woof", "neigh", "", "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.TransferAsset(context.Background(), "spot", "p2p", testCrypto.String(), testPair.String(),
		clientIDGenerator(), testAmount)
	assert.NoError(t, err)
}

func TestGetTransferableCoinList(t *testing.T) {
	t.Parallel()
	_, err := bi.GetTransferableCoinList(context.Background(), "", "")
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.GetTransferableCoinList(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errToTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetTransferableCoinList(context.Background(), "spot", "p2p")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestSubaccountTransfer(t *testing.T) {
	t.Parallel()
	_, err := bi.SubaccountTransfer(context.Background(), "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.SubaccountTransfer(context.Background(), "meow", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errToTypeEmpty)
	_, err = bi.SubaccountTransfer(context.Background(), "meow", "woof", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errCurrencyAndPairEmpty)
	_, err = bi.SubaccountTransfer(context.Background(), "meow", "woof", "neigh", "", "", "", "", 0)
	assert.ErrorIs(t, err, errFromIDEmpty)
	_, err = bi.SubaccountTransfer(context.Background(), "meow", "woof", "neigh", "", "", "oink", "", 0)
	assert.ErrorIs(t, err, errToIDEmpty)
	_, err = bi.SubaccountTransfer(context.Background(), "meow", "woof", "neigh", "", "", "oink", "quack", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	fromID := subAccTestHelper(t, "", strings.ToLower(string(testSubaccountName[:3]))+"****@virtual-bitget.com")
	toID := subAccTestHelper(t, strings.ToLower(string(testSubaccountName[:3]))+"****@virtual-bitget.com", "")
	_, err = bi.SubaccountTransfer(context.Background(), "spot", "p2p", testCrypto.String(), testPair.String(),
		clientIDGenerator(), fromID, toID, testAmount)
	assert.NoError(t, err)
}

func TestWithdrawFunds(t *testing.T) {
	t.Parallel()
	_, err := bi.WithdrawFunds(context.Background(), "", "", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.WithdrawFunds(context.Background(), "meow", "", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errTransferTypeEmpty)
	_, err = bi.WithdrawFunds(context.Background(), "meow", "woof", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errAddressEmpty)
	_, err = bi.WithdrawFunds(context.Background(), "meow", "woof", "neigh", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.WithdrawFunds(context.Background(), testCrypto.String(), "on_chain", testAddress, testCrypto.String(), "",
		"", "", "", clientIDGenerator(), testAmount)
	assert.NoError(t, err)
}

func TestGetSubaccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSubaccountTransferRecord(context.Background(), "", "", "", time.Now().Add(time.Hour), time.Time{}, 0,
		0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSubaccountTransferRecord(context.Background(), "", "", "meow", time.Time{}, time.Time{}, 3, 1<<62)
	assert.NoError(t, err)
}

func TestGetTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := bi.GetTransferRecord(context.Background(), "", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.GetTransferRecord(context.Background(), "meow", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.GetTransferRecord(context.Background(), "meow", "woof", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetTransferRecord(context.Background(), testCrypto.String(), "spot", "meow", time.Time{}, time.Time{},
		3, 1<<62)
	assert.NoError(t, err)
}

func TestSwitchBGBDeductionStatus(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.SwitchBGBDeductionStatus, false, false, nil, false, true, canManipulateRealOrders)
	testGetOneArg(t, bi.SwitchBGBDeductionStatus, false, true, nil, false, true, canManipulateRealOrders)
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	_, err := bi.GetDepositAddressForCurrency(context.Background(), "", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetDepositAddressForCurrency(context.Background(), testCrypto.String(), "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSubaccountDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSubaccountDepositAddress(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.GetSubaccountDepositAddress(context.Background(), "meow", "", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSubaccountDepositAddress(context.Background(), deposSubaccID, testCrypto.String(), "")
	// Getting the error "The business of this account has been restricted", don't think this was happening
	// last week.
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetBGBDeductionStatus(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetBGBDeductionStatus)
}

func TestGetSubaccountDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSubaccountDepositRecords(context.Background(), "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.GetSubaccountDepositRecords(context.Background(), "meow", "", 0, 0, 0, time.Now().Add(time.Hour),
		time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := subAccTestHelper(t, "", "")
	_, err = bi.GetSubaccountDepositRecords(context.Background(), tarID, "", 0, 1<<62, 2, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetWithdrawalRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetWithdrawalRecords(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetWithdrawalRecords(context.Background(), "", "", time.Now().Add(-time.Hour*24*90), time.Now(),
		1<<62, 0, 5)
	assert.NoError(t, err)
}

func TestGetDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetDepositRecords(context.Background(), "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetDepositRecords(context.Background(), testCrypto.String(), 0, 1<<62, 2,
		time.Now().Add(-time.Hour*24*90), time.Now())
	assert.NoError(t, err)
}

func TestGetFuturesMergeDepth(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesMergeDepth(context.Background(), "", "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesMergeDepth(context.Background(), "meow", "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetFuturesMergeDepth(context.Background(), testPair.String(), "USDT-FUTURES", "scale3", "5")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetFuturesVIPFeeRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetFuturesVIPFeeRate)
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
	_, err := bi.GetRecentFuturesFills(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetRecentFuturesFills(context.Background(), "meow", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetRecentFuturesFills(context.Background(), testPair.String(), "USDT-FUTURES", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetFuturesMarketTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesMarketTrades(context.Background(), "", "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesMarketTrades(context.Background(), "meow", "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesMarketTrades(context.Background(), "meow", "woof", 0, 0, time.Now().Add(time.Hour),
		time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := bi.GetFuturesMarketTrades(context.Background(), testPair.String(), "USDT-FUTURES", 5, 1<<62,
		time.Time{}, time.Time{})
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetFuturesCandlestickData(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesCandlestickData(context.Background(), "", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesCandlestickData(context.Background(), "meow", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesCandlestickData(context.Background(), "meow", "woof", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errGranEmpty)
	_, err = bi.GetFuturesCandlestickData(context.Background(), "meow", "woof", "neigh", time.Now().Add(time.Hour),
		time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	resp, err := bi.GetFuturesCandlestickData(context.Background(), testPair.String(), "USDT-FUTURES", "1m",
		time.Time{}, time.Time{}, 5, CallModeNormal)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.FuturesCandles)
	resp, err = bi.GetFuturesCandlestickData(context.Background(), testPair.String(), "COIN-FUTURES", "1m",
		time.Time{}, time.Time{}, 5, CallModeHistory)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.FuturesCandles)
	resp, err = bi.GetFuturesCandlestickData(context.Background(), testPair.String(), "USDC-FUTURES", "1m",
		time.Time{}, time.Now(), 5, CallModeIndex)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.FuturesCandles)
	resp, err = bi.GetFuturesCandlestickData(context.Background(), testPair.String(), "USDT-FUTURES", "1m",
		time.Time{}, time.Now(), 5, CallModeMark)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.FuturesCandles)
}

func TestGetOpenPositions(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, bi.GetOpenPositions)
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
	_, err := bi.GetFundingHistorical(context.Background(), "", "", 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFundingHistorical(context.Background(), "meow", "", 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetFundingHistorical(context.Background(), testPair.String(), "USDT-FUTURES", 5, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetFundingCurrent(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, bi.GetFundingCurrent)
}

func TestGetContractConfig(t *testing.T) {
	t.Parallel()
	testGetTwoArgs(t, bi.GetContractConfig)
}

func TestGetOneFuturesAccount(t *testing.T) {
	t.Parallel()
	_, err := bi.GetOneFuturesAccount(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetOneFuturesAccount(context.Background(), "meow", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetOneFuturesAccount(context.Background(), "meow", "woof", "")
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetOneFuturesAccount(context.Background(), testPair.String(), "USDT-FUTURES", "USDT")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetAllFuturesAccounts(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetAllFuturesAccounts, "", "COIN-FUTURES", errProductTypeEmpty, true, true, true)
}

func TestGetFuturesSubaccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetFuturesSubaccountAssets, "", "COIN-FUTURES", errProductTypeEmpty, true, true, true)
}

func TestGetEstimatedOpenCount(t *testing.T) {
	t.Parallel()
	_, err := bi.GetEstimatedOpenCount(context.Background(), "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetEstimatedOpenCount(context.Background(), "meow", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetEstimatedOpenCount(context.Background(), "meow", "woof", "", 0, 0, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.GetEstimatedOpenCount(context.Background(), "meow", "woof", "neigh", 0, 0, 0)
	assert.ErrorIs(t, err, errOpenAmountEmpty)
	_, err = bi.GetEstimatedOpenCount(context.Background(), "meow", "woof", "neigh", 1, 0, 0)
	assert.ErrorIs(t, err, errOpenPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetEstimatedOpenCount(context.Background(), testPair.String(), "USDT-FUTURES", "USDT",
		testPrice, testAmount, 20)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestChangeLeverage(t *testing.T) {
	t.Parallel()
	_, err := bi.ChangeLeverage(context.Background(), "", "", "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.ChangeLeverage(context.Background(), "meow", "", "", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ChangeLeverage(context.Background(), "meow", "woof", "", "", 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.ChangeLeverage(context.Background(), "meow", "woof", "neigh", "", 0)
	assert.ErrorIs(t, err, errLeverageEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ChangeLeverage(context.Background(), testPair.String(), "USDT-FUTURES", "USDT", "", 20)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestAdjustMargin(t *testing.T) {
	t.Parallel()
	err := bi.AdjustMargin(context.Background(), "", "", "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	err = bi.AdjustMargin(context.Background(), "meow", "", "", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	err = bi.AdjustMargin(context.Background(), "meow", "woof", "", "", 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	err = bi.AdjustMargin(context.Background(), "meow", "woof", "neigh", "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	// This is getting the error "verification exception margin mode == FIXED", and I can't find a way to
	// skirt around that
	err = bi.AdjustMargin(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES", testFiat2.String(),
		"long", -testAmount)
	assert.NoError(t, err)
}

func TestChangeMarginMode(t *testing.T) {
	t.Parallel()
	_, err := bi.ChangeMarginMode(context.Background(), "", "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.ChangeMarginMode(context.Background(), "meow", "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ChangeMarginMode(context.Background(), "meow", "woof", "", "")
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.ChangeMarginMode(context.Background(), "meow", "woof", "neigh", "")
	assert.ErrorIs(t, err, errMarginModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.ChangeMarginMode(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES",
		testFiat2.String(), "crossed")
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
	_, err := bi.ChangePositionMode(context.Background(), "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ChangePositionMode(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errPositionModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ChangePositionMode(context.Background(), testFiat2.String()+"-FUTURES", "hedge_mode")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetFuturesAccountBills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesAccountBills(context.Background(), "", "", "", "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesAccountBills(context.Background(), "meow", "", "", "", 0, 0, time.Now().Add(time.Hour),
		time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesAccountBills(context.Background(), testFiat2.String()+"-FUTURES", "", "", "", 0, 0,
		time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetPositionTier(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPositionTier(context.Background(), "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetPositionTier(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetPositionTier(context.Background(), testFiat2.String()+"-FUTURES", testPair2.String())
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSinglePosition(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSinglePosition(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetSinglePosition(context.Background(), "meow", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSinglePosition(context.Background(), "meow", "woof", "")
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSinglePosition(context.Background(), testFiat2.String()+"-FUTURES", testPair2.String(),
		testFiat2.String())
	assert.NoError(t, err)
}

func TestGetAllPositions(t *testing.T) {
	t.Parallel()
	_, err := bi.GetAllPositions(context.Background(), "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetAllPositions(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetAllPositions(context.Background(), testFiat2.String()+"-FUTURES", testFiat2.String())
	assert.NoError(t, err)
}

func TestGetHistoricalPositions(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalPositions(context.Background(), "", "", 0, 0, time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetHistoricalPositions(context.Background(), "", "", 1<<62, 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceFuturesOrder(context.Background(), "", "", "", "", "", "", "", "", "", 0, 0, 0, 0, false,
		false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "", "", "", "", "", "", "", "", 0, 0, 0, 0, false,
		false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "", "", "", "", "", "", "", 0, 0, 0, 0,
		false, false)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "", "", "", "", "", "", 0, 0, 0,
		0, false, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "", "", "", "", 0, 0,
		0, 0, false, false)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "", "", "", "",
		0, 0, 0, 0, false, false)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "", "limit", "",
		"", 0, 0, 0, 0, false, false)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "", "limit", "",
		"", 0, 0, 1, 0, false, false)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.PlaceFuturesOrder(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES",
		"isolated", testFiat2.String(), "buy", "open", "limit", "GTC", clientIDGenerator(), testPrice2+1, testPrice2-1,
		testAmount2, testPrice2, true, true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestPlaceReversal(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceReversal(context.Background(), "", "", "", "", "", "", 0, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceReversal(context.Background(), "meow", "", "", "", "", "", 0, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceReversal(context.Background(), "meow", "woof", "", "", "", "", 0, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceReversal(context.Background(), "meow", "woof", "neigh", "", "", "", 0, false)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceReversal(context.Background(), testPair2.String(), testFiat2.String(),
		testFiat2.String()+"-FUTURES", "Buy", "Open", clientIDGenerator(), testAmount, true)
	assert.NoError(t, err)
}

func TestBatchPlaceFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.BatchPlaceFuturesOrders(context.Background(), "", "", "", "", nil, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchPlaceFuturesOrders(context.Background(), "meow", "", "", "", nil, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.BatchPlaceFuturesOrders(context.Background(), "meow", "woof", "", "", nil, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.BatchPlaceFuturesOrders(context.Background(), "meow", "woof", "neigh", "", nil, false)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	_, err = bi.BatchPlaceFuturesOrders(context.Background(), "meow", "woof", "neigh", "oink", nil, false)
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
	_, err = bi.BatchPlaceFuturesOrders(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES",
		testFiat2.String(), "isolated", orders, true)
	assert.NoError(t, err)
}

func TestGetFuturesOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesOrderDetails(context.Background(), "", "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesOrderDetails(context.Background(), "meow", "", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesOrderDetails(context.Background(), "meow", "woof", "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	oID := getFuturesOrdIDHelper(t, false, true)
	_, err = bi.GetFuturesOrderDetails(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES",
		oID.ClientOrderID, oID.OrderID)
	assert.NoError(t, err)
}

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesFills(context.Background(), 0, 0, 0, "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesFills(context.Background(), 0, 0, 0, "", "meow", time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesFills(context.Background(), 0, 1<<62, 5, "", testFiat2.String()+"-FUTURES", time.Time{},
		time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesOrderFillHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesOrderFillHistory(context.Background(), "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesOrderFillHistory(context.Background(), "", "meow", 0, 0, 0, time.Now().Add(time.Hour),
		time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// Keeps getting "Parameter verification failed" error and I can't figure out why
	resp, err := bi.GetFuturesOrderFillHistory(context.Background(), "", testFiat2.String()+"-FUTURES", 0, 1<<62, 5,
		time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.Data.FillList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetFuturesOrderFillHistory(context.Background(), "", testFiat2.String()+"-FUTURES",
		resp.Data.FillList[0].OrderID, 1<<62, 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetPendingFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPendingFuturesOrders(context.Background(), 0, 0, 0, "", "", "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetPendingFuturesOrders(context.Background(), 0, 0, 0, "", "", "meow", "", time.Now().Add(time.Hour),
		time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetPendingFuturesOrders(context.Background(), 0, 1<<62, 5, "", testPair2.String(),
		testFiat2.String()+"-FUTURES", "", time.Now().Add(-time.Hour*24*90), time.Now())
	require.NoError(t, err)
	if len(resp.Data.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetPendingFuturesOrders(context.Background(), resp.Data.EntrustedList[0].OrderID, 1<<62, 5, "",
		testPair2.String(), testFiat2.String()+"-FUTURES", "", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetHistoricalFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalFuturesOrders(context.Background(), 0, 0, 0, "", "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetHistoricalFuturesOrders(context.Background(), 0, 0, 0, "", "", "meow", time.Now().Add(time.Hour),
		time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetHistoricalFuturesOrders(context.Background(), 0, 1<<62, 5, "", testPair2.String(),
		testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.Data.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetHistoricalFuturesOrders(context.Background(), resp.Data.EntrustedList[0].OrderID, 1<<62, 5, "",
		testPair2.String(), testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetFuturesTriggerOrderByID(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesTriggerOrderByID(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.GetFuturesTriggerOrderByID(context.Background(), "meow", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesTriggerOrderByID(context.Background(), "meow", "woof", 0)
	assert.ErrorIs(t, err, errPlanOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetHistoricalTriggerFuturesOrders(context.Background(), 0, 1<<62, 5, "", "normal_plan", "",
		testPair2.String(), testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.Data.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetFuturesTriggerOrderByID(context.Background(), "normal_plan", testFiat2.String()+"-FUTURES",
		resp.Data.EntrustedList[0].OrderID)
	assert.NoError(t, err)
}

func TestPlaceTPSLFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceTPSLFuturesOrder(context.Background(), "", "", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(context.Background(), "meow", "", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(context.Background(), "meow", "woof", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(context.Background(), "meow", "woof", "neigh", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "", "", "", 0,
		0, 0)
	assert.ErrorIs(t, err, errHoldSideEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "quack", "", "",
		0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.PlaceTPSLFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "quack", "", "",
		1, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.PlaceTPSLFuturesOrder(context.Background(), testFiat2.String(), testFiat2.String()+"-FUTURES",
		testPair2.String(), "profit_plan", "", "short", "", "", testPrice2+2, 0, testAmount2)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestPlaceTriggerFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceTriggerFuturesOrder(context.Background(), "", "", "", "", "", "", "", "", "", "", "", "", 0, 0,
		0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "", "", "", "", "", "", "", "", "", "", "", 0,
		0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "", "", "", "", "", "", "", "", "", "",
		0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "", "", "", "", "", "", "",
		"", "", 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "", "", "", "",
		"", "", "", 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "", "", "",
		"", "", "", "", 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errTriggerTypeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "baa", "",
		"", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "baa",
		"moo", "", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "baa",
		"moo", "", "cluck", "", "", "", 0, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "baa",
		"moo", "", "cluck", "", "", "", 1, 0, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errExecutePriceEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "baa",
		"moo", "", "cluck", "", "", "", 1, 1, 0, 0, 0, 0, 0, 0, false)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "baa",
		"moo", "", "cluck", "", "", "", 1, 1, 0, 1, 1, 0, 0, 0, false)
	assert.ErrorIs(t, err, errTakeProfitParamsInconsistency)
	_, err = bi.PlaceTriggerFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "baa",
		"moo", "", "cluck", "", "", "", 1, 1, 0, 1, 0, 0, 1, 0, false)
	assert.ErrorIs(t, err, errStopLossParamsInconsistency)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	// This returns the error "The parameter does not meet the specification d delegateType is error". The
	// documentation doesn't mention that parameter anywhere, nothing seems similar to it, and attempts to send
	// that parameter with various values, or to tweak other parameters, yielded no difference
	resp, err := bi.PlaceTriggerFuturesOrder(context.Background(), "normal_plan", testPair2.String(),
		testFiat2.String()+"-FUTURES", "isolated", testFiat2.String(), "mark_price", "Sell", "", "limit",
		clientIDGenerator(), "", "", testAmount2*1000, testPrice2+2, 0, testPrice2+1, 0, 0, 0, 0, false)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestModifyTPSLFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.ModifyTPSLFuturesOrder(context.Background(), 0, "", "", "", "", "", 0, 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(context.Background(), 1, "", "", "", "", "", 0, 0, 0, 0)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(context.Background(), 1, "", "meow", "", "", "", 0, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(context.Background(), 1, "", "meow", "woof", "", "", 0, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(context.Background(), 1, "", "meow", "woof", "neigh", "", 0, 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.ModifyTPSLFuturesOrder(context.Background(), 1, "", "meow", "woof", "neigh", "", 1, 0, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ID := getTrigOrdIDHelper(t, []string{"profit_loss"})
	resp, err := bi.ModifyTPSLFuturesOrder(context.Background(), ID.OrderID, ID.ClientOrderID, testFiat2.String(),
		testFiat2.String()+"-FUTURES", testPair2.String(), "", testPrice2-1, testPrice2+2, testAmount2, 0.1)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestModifyTriggerFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.ModifyTriggerFuturesOrder(context.Background(), 0, "", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.ModifyTriggerFuturesOrder(context.Background(), 1, "", "", "", "", "", 0, 0, 0, 0, 0, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_ = getTrigOrdIDHelper(t, []string{"normal_plan", "track_plan"})
	t.Skip("TODO: Finish once PlaceTriggerFuturesOrder is fixed")
}

func TestGetPendingTriggerFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPendingTriggerFuturesOrders(context.Background(), 0, 0, 0, "", "", "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.GetPendingTriggerFuturesOrders(context.Background(), 0, 0, 0, "", "", "meow", "", time.Time{},
		time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetPendingTriggerFuturesOrders(context.Background(), 0, 0, 0, "", "", "meow", "woof",
		time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetPendingTriggerFuturesOrders(context.Background(), 0, 1<<62, 5, "", testPair2.String(),
		"profit_loss", testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.Data.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetPendingTriggerFuturesOrders(context.Background(), resp.Data.EntrustedList[0].OrderID, 1<<62, 5, "",
		testPair2.String(), "profit_loss", testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetHistoricalTriggerFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalTriggerFuturesOrders(context.Background(), 0, 0, 0, "", "", "", "", "", time.Time{},
		time.Time{})
	assert.ErrorIs(t, err, errPlanTypeEmpty)
	_, err = bi.GetHistoricalTriggerFuturesOrders(context.Background(), 0, 0, 0, "", "meow", "", "", "", time.Time{},
		time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetHistoricalTriggerFuturesOrders(context.Background(), 0, 0, 0, "", "meow", "", "", "woof",
		time.Now().Add(time.Hour), time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetHistoricalTriggerFuturesOrders(context.Background(), 0, 1<<62, 5, "", "normal_plan", "",
		testPair2.String(), testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
	require.NoError(t, err)
	if len(resp.Data.EntrustedList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetHistoricalTriggerFuturesOrders(context.Background(), resp.Data.EntrustedList[0].OrderID, 1<<62, 5,
		"", "normal_plan", "", testPair2.String(), testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetSupportedCurrencies(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetSupportedCurrencies)
}

func TestGetCrossBorrowHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossBorrowHistory(context.Background(), 0, 0, 0, "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossBorrowHistory(context.Background(), 1, 2, 1<<62, "", time.Now().Add(-time.Hour*24*85),
		time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossRepayHistory(context.Background(), 0, 0, 0, "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossRepayHistory(context.Background(), 1, 2, 1<<62, "", time.Now().Add(-time.Hour*24*85),
		time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossInterestHistory(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossInterestHistory(context.Background(), "", time.Now().Add(-time.Hour*24*85), time.Time{}, 2,
		1<<62)
	assert.NoError(t, err)
}

func TestGetCrossLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossLiquidationHistory(context.Background(), time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossLiquidationHistory(context.Background(), time.Now().Add(-time.Hour*24*85), time.Time{}, 2,
		1<<62)
	assert.NoError(t, err)
}

func TestGetCrossFinancialHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossFinancialHistory(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossFinancialHistory(context.Background(), "", "", time.Now().Add(-time.Hour*24*85), time.Time{},
		2, 1<<62)
	assert.NoError(t, err)
}

func TestGetCrossAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossAccountAssets, "", "", nil, false, true, true)
}

func TestCrossBorrow(t *testing.T) {
	t.Parallel()
	_, err := bi.CrossBorrow(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.CrossBorrow(context.Background(), "meow", "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CrossBorrow(context.Background(), testFiat.String(), clientIDGenerator(), testAmount)
	assert.NoError(t, err)
}

func TestCrossRepay(t *testing.T) {
	t.Parallel()
	_, err := bi.CrossRepay(context.Background(), "", 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.CrossRepay(context.Background(), "meow", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CrossRepay(context.Background(), testFiat.String(), testAmount)
	assert.NoError(t, err)
}

func TestGetCrossRiskRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetCrossRiskRate)
}

func TestGetCrossMaxBorrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossMaxBorrowable, "", testFiat.String(), errCurrencyEmpty, false, true, true)
}

func TestGetCrossMaxTransferable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossMaxTransferable, "", testFiat.String(), errCurrencyEmpty, false, true, true)
}

func TestGetCrossInterestRateAndMaxBorrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossInterestRateAndMaxBorrowable, "", testFiat.String(), errCurrencyEmpty, false, true,
		true)
}

func TestGetCrossTierConfiguration(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetCrossTierConfiguration, "", testFiat.String(), errCurrencyEmpty, false, true, true)
}

func TestCrossFlashRepay(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.CrossFlashRepay, "", testFiat.String(), nil, false, true, canManipulateRealOrders)
}

func TestGetCrossFlashRepayResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossFlashRepayResult(context.Background(), nil)
	assert.ErrorIs(t, err, errIDListEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	// This must be done, as this is the only way to get a repayment ID
	resp, err := bi.CrossFlashRepay(context.Background(), testFiat.String())
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data)
	_, err = bi.GetCrossFlashRepayResult(context.Background(), []int64{resp.Data.RepayID})
	assert.NoError(t, err)
}

func TestPlaceCrossOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceCrossOrder(context.Background(), "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceCrossOrder(context.Background(), "meow", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceCrossOrder(context.Background(), "meow", "woof", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errLoanTypeEmpty)
	_, err = bi.PlaceCrossOrder(context.Background(), "meow", "woof", "neigh", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errStrategyEmpty)
	_, err = bi.PlaceCrossOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceCrossOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "quack", 0, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceCrossOrder(context.Background(), testPair.String(), "limit", "normal", "GTC", "", "Buy",
		testPrice2, testAmount2, 0)
	assert.NoError(t, err)
}

func TestBatchPlaceCrossOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.BatchPlaceCrossOrders(context.Background(), "", nil)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchPlaceCrossOrders(context.Background(), "meow", nil)
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
	_, err = bi.BatchPlaceCrossOrders(context.Background(), testPair.String(), orders)
	assert.NoError(t, err)
}

func TestGetCrossOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossOpenOrders(context.Background(), "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCrossOpenOrders(context.Background(), "meow", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp := getCrossOrdIDHelper(t, true)
	_, err = bi.GetCrossOpenOrders(context.Background(), testPair.String(), "", resp.OrderID,
		5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossHistoricalorders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossHistoricalOrders(context.Background(), "", "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCrossHistoricalOrders(context.Background(), "meow", "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetCrossHistoricalOrders(context.Background(), testPair.String(), "", "", 0, 5, 1<<62,
		time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Data.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetCrossHistoricalOrders(context.Background(), testPair.String(), "", "",
		resp.Data.OrderList[0].OrderID, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossOrderFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossOrderFills(context.Background(), "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCrossOrderFills(context.Background(), "meow", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetCrossOrderFills(context.Background(), testPair.String(), 0, 1<<62, 5,
		time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Data.Fills) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetCrossOrderFills(context.Background(), testPair.String(), resp.Data.Fills[0].OrderID, 1<<62, 5,
		time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetCrossLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCrossLiquidationOrders(context.Background(), "", "", "", "", time.Now().Add(time.Hour), time.Time{},
		5, 1<<62)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetCrossLiquidationOrders(context.Background(), "", "", "", "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedRepayHistory(context.Background(), "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedRepayHistory(context.Background(), "meow", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetIsolatedRepayHistory(context.Background(), testPair.String(), "", 0, 5, 1<<62,
		time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Data.ResultList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetIsolatedRepayHistory(context.Background(), testPair.String(), "", resp.Data.ResultList[0].RepayID,
		5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedInterestHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedInterestHistory(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedInterestHistory(context.Background(), "meow", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedInterestHistory(context.Background(), testPair.String(), "",
		time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedLiquidationHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedLiquidationHistory(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedLiquidationHistory(context.Background(), "meow", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedLiquidationHistory(context.Background(), testPair.String(),
		time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedFinancialHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedFinancialHistory(context.Background(), "", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedFinancialHistory(context.Background(), "meow", "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedFinancialHistory(context.Background(), testPair.String(), "", "",
		time.Now().Add(-time.Hour*24*85), time.Time{}, 2, 1<<62)
	assert.NoError(t, err)
}

func TestGetIsolatedAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetIsolatedAccountAssets, "", "", nil, false, true, true)
}

func TestIsolatedBorrow(t *testing.T) {
	t.Parallel()
	_, err := bi.IsolatedBorrow(context.Background(), "", "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.IsolatedBorrow(context.Background(), "meow", "", "", 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.IsolatedBorrow(context.Background(), "meow", "woof", "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.IsolatedBorrow(context.Background(), testPair.String(), testFiat.String(), "", testAmount)
	assert.NoError(t, err)
}

func TestIsolatedRepay(t *testing.T) {
	t.Parallel()
	_, err := bi.IsolatedRepay(context.Background(), 0, "", "", "")
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.IsolatedRepay(context.Background(), 1, "", "", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.IsolatedRepay(context.Background(), 1, "meow", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.IsolatedRepay(context.Background(), testAmount, testFiat.String(), testPair.String(), "")
	assert.NoError(t, err)
}

func TestGetIsolatedRiskRate(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetIsolatedRiskRate(context.Background(), "", 1, 5)
	assert.NoError(t, err)
}

func TestGetIsolatedInterestRateAndMaxBorrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetIsolatedInterestRateAndMaxBorrowable, "", testPair.String(), errPairEmpty, false, true,
		true)
}

func TestGetIsolatedMaxborrowable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetIsolatedMaxBorrowable, "", testPair.String(), errPairEmpty, false, true, true)
}

func TestGetIsolatedMaxTransferable(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetIsolatedMaxTransferable, "", testPair.String(), errPairEmpty, false, true, true)
}

func TestIsolatedFlashRepay(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.IsolatedFlashRepay, nil, []string{testPair.String()}, nil, false, true,
		canManipulateRealOrders)
}

func TestGetIsolatedFlashRepayResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedFlashRepayResult(context.Background(), nil)
	assert.ErrorIs(t, err, errIDListEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	// This must be done, as this is the only way to get a repayment ID
	resp, err := bi.IsolatedFlashRepay(context.Background(), []string{testPair.String()})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data)
	_, err = bi.GetIsolatedFlashRepayResult(context.Background(), []int64{resp.Data[0].RepayID})
	assert.NoError(t, err)
}

func TestPlaceIsolatedOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceIsolatedOrder(context.Background(), "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceIsolatedOrder(context.Background(), "meow", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceIsolatedOrder(context.Background(), "meow", "woof", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errLoanTypeEmpty)
	_, err = bi.PlaceIsolatedOrder(context.Background(), "meow", "woof", "neigh", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errStrategyEmpty)
	_, err = bi.PlaceIsolatedOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceIsolatedOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "quack", 0, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceIsolatedOrder(context.Background(), testPair.String(), "limit", "normal", "GTC", "", "Buy",
		testPrice2, testAmount2, 0)
	assert.NoError(t, err)
}

func TestBatchPlaceIsolatedOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.BatchPlaceIsolatedOrders(context.Background(), "", nil)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchPlaceIsolatedOrders(context.Background(), "meow", nil)
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
	_, err = bi.BatchPlaceIsolatedOrders(context.Background(), testPair.String(), orders)
	assert.NoError(t, err)
}

func TestGetIsolatedOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedOpenOrders(context.Background(), "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedOpenOrders(context.Background(), "meow", "", 0, 0, 0, time.Now().Add(time.Hour),
		time.Time{})
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp := getIsoOrdIDHelper(t, true)
	_, err = bi.GetIsolatedOpenOrders(context.Background(), testPair.String(), "", resp.OrderID,
		5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedHistoricalOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedHistoricalOrders(context.Background(), "", "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedHistoricalOrders(context.Background(), "meow", "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetIsolatedHistoricalOrders(context.Background(), testPair.String(), "", "", 0, 5, 1<<62,
		time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Data.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetIsolatedHistoricalOrders(context.Background(), testPair.String(), "", "",
		resp.Data.OrderList[0].OrderID, 5, 1<<62, time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedOrderFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedOrderFills(context.Background(), "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetIsolatedOrderFills(context.Background(), "meow", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetIsolatedOrderFills(context.Background(), testPair.String(), 0, 1<<62, 5,
		time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Data.Fills) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetIsolatedOrderFills(context.Background(), testPair.String(), resp.Data.Fills[0].OrderID, 1<<62, 5,
		time.Now().Add(-time.Hour*24*85), time.Time{})
	assert.NoError(t, err)
}

func TestGetIsolatedLiquidationOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetIsolatedLiquidationOrders(context.Background(), "", "", "", "", time.Now().Add(time.Hour),
		time.Time{}, 5, 1<<62)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetIsolatedLiquidationOrders(context.Background(), "", "", "", "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSavingsProductList(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsProductList(context.Background(), "", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSavingsProductList(context.Background(), testCrypto.String(), "")
	assert.NoError(t, err)
}

func TestGetSavingsBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetSavingsBalance)
}

func TestGetSavingsAssets(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsAssets(context.Background(), "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSavingsAssets(context.Background(), "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSavingsRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsRecords(context.Background(), "", "", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSavingsRecords(context.Background(), "", "", "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSavingsSubscriptionDetail(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsSubscriptionDetail(context.Background(), 0, "")
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = bi.GetSavingsSubscriptionDetail(context.Background(), 1, "")
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSavingsProductList(context.Background(), testCrypto.String(), "")
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data)
	_, err = bi.GetSavingsSubscriptionDetail(context.Background(), resp.Data[0].ProductID, resp.Data[0].PeriodType)
	assert.NoError(t, err)
}

func TestSubscribeSavings(t *testing.T) {
	t.Parallel()
	_, err := bi.SubscribeSavings(context.Background(), 0, "", 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = bi.SubscribeSavings(context.Background(), 1, "", 0)
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	_, err = bi.SubscribeSavings(context.Background(), 1, "meow", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetSavingsProductList(context.Background(), testCrypto.String(), "")
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data)
	resp2, err := bi.GetSavingsSubscriptionDetail(context.Background(), resp.Data[0].ProductID, resp.Data[0].PeriodType)
	require.NoError(t, err)
	require.NotEmpty(t, resp2.Data)
	_, err = bi.SubscribeSavings(context.Background(), resp.Data[0].ProductID, resp.Data[0].PeriodType,
		resp2.Data.SingleMinAmount)
	assert.NoError(t, err)
}

func TestGetSavingsSubscriptionResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsSubscriptionResult(context.Background(), 0, "")
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = bi.GetSavingsSubscriptionResult(context.Background(), 1, "")
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSavingsRecords(context.Background(), "", "", "", time.Time{}, time.Time{}, 100, 0)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data.ResultList)
	tarID := -1
	for x := range resp.Data.ResultList {
		if resp.Data.ResultList[x].OrderType == "subscribe" {
			tarID = x
			break
		}
	}
	if tarID == -1 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetSavingsSubscriptionResult(context.Background(), resp.Data.ResultList[tarID].OrderID,
		resp.Data.ResultList[tarID].ProductType)
	assert.NoError(t, err)
}

func TestRedeemSavings(t *testing.T) {
	t.Parallel()
	_, err := bi.RedeemSavings(context.Background(), 0, 0, "", 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = bi.RedeemSavings(context.Background(), 1, 0, "", 0)
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	_, err = bi.RedeemSavings(context.Background(), 1, 0, "meow", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	var pagination int64
	var tarProd int64
	var tarOrd int64
	var tarPeriod string
	for {
		resp, err := bi.GetSavingsAssets(context.Background(), "", time.Time{}, time.Time{}, 100, pagination)
		require.NoError(t, err)
		require.NotEmpty(t, resp.Data)
		for i := range resp.Data.ResultList {
			if resp.Data.ResultList[i].AllowRedemption && resp.Data.ResultList[i].Status != "in_redemption" {
				tarProd = resp.Data.ResultList[i].ProductID
				tarOrd = resp.Data.ResultList[i].OrderID
				tarPeriod = resp.Data.ResultList[i].PeriodType
				break
			}
		}
		if tarProd != 0 || int64(resp.Data.EndID) == pagination || resp.Data.EndID == 0 {
			break
		} else {
			pagination = int64(resp.Data.EndID)
		}
	}
	if tarProd == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.RedeemSavings(context.Background(), tarProd, tarOrd, tarPeriod, testAmount)
	assert.NoError(t, err)
}

func TestGetSavingsRedemptionResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSavingsRedemptionResult(context.Background(), 0, "")
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = bi.GetSavingsRedemptionResult(context.Background(), 1, "")
	assert.ErrorIs(t, err, errPeriodTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSavingsRecords(context.Background(), "", "", "", time.Time{}, time.Time{}, 100, 0)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data.ResultList)
	tarID := -1
	for x := range resp.Data.ResultList {
		if resp.Data.ResultList[x].OrderType == "redeem" {
			tarID = x
			break
		}
	}
	if tarID == -1 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetSavingsRedemptionResult(context.Background(), resp.Data.ResultList[tarID].OrderID,
		resp.Data.ResultList[tarID].ProductType)
	assert.NoError(t, err)
}

func TestGetEarnAccountAssets(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetEarnAccountAssets, "", "", nil, false, true, true)
}

func TestGetSharkFinProducts(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinProducts(context.Background(), "", 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSharkFinProducts(context.Background(), testCrypto.String(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSharkFinBalance(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetSharkFinBalance)
}

func TestGetSharkFinAssets(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinAssets(context.Background(), "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSharkFinAssets(context.Background(), "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSharkFinRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinRecords(context.Background(), "", "", time.Now().Add(time.Hour), time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterTimeNow)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSharkFinRecords(context.Background(), "", "", time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetSharkFinSubscriptionDetail(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinSubscriptionDetail(context.Background(), 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSharkFinProducts(context.Background(), testCrypto.String(), 5, 1<<62)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data)
	_, err = bi.GetSharkFinSubscriptionDetail(context.Background(), resp.Data.ResultList[0].ProductID)
	assert.NoError(t, err)
}

func TestSubscribeSharkFin(t *testing.T) {
	t.Parallel()
	_, err := bi.SubscribeSharkFin(context.Background(), 0, 0)
	assert.ErrorIs(t, err, errProductIDEmpty)
	_, err = bi.SubscribeSharkFin(context.Background(), 1, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetSharkFinProducts(context.Background(), testCrypto.String(), 5, 1<<62)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data)
	_, err = bi.SubscribeSharkFin(context.Background(), resp.Data.ResultList[0].ProductID,
		resp.Data.ResultList[0].MinAmount)
	assert.NoError(t, err)
}

func TestGetSharkFinSubscriptionResult(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSharkFinSubscriptionResult(context.Background(), 0)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSharkFinRecords(context.Background(), "", "", time.Time{}, time.Time{}, 100, 0)
	require.NoError(t, err)
	require.NotEmpty(t, resp.Data.ResultList)
	tarID := -1
	for x := range resp.Data.ResultList {
		if resp.Data.ResultList[x].Type == "subscribe" {
			tarID = x
			break
		}
	}
	if tarID == -1 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetSharkFinSubscriptionResult(context.Background(), resp.Data.ResultList[tarID].OrderID)
	assert.NoError(t, err)
}

func TestGetLoanCurrencyList(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetLoanCurrencyList, "", testFiat.String(), errCurrencyEmpty, false, true, true)
}

func TestGetEstimatedInterestAndBorrowable(t *testing.T) {
	t.Parallel()
	_, err := bi.GetEstimatedInterestAndBorrowable(context.Background(), "", "", "", 0)
	assert.ErrorIs(t, err, errLoanCoinEmpty)
	_, err = bi.GetEstimatedInterestAndBorrowable(context.Background(), "meow", "", "", 0)
	assert.ErrorIs(t, err, errCollateralCoinEmpty)
	_, err = bi.GetEstimatedInterestAndBorrowable(context.Background(), "meow", "woof", "", 0)
	assert.ErrorIs(t, err, errTermEmpty)
	_, err = bi.GetEstimatedInterestAndBorrowable(context.Background(), "meow", "woof", "neigh", 0)
	assert.ErrorIs(t, err, errCollateralAmountEmpty)
	_, err = bi.GetEstimatedInterestAndBorrowable(context.Background(), testCrypto.String(), testFiat.String(), "SEVEN",
		testPrice)
	assert.NoError(t, err)
}

func TestBorrowFunds(t *testing.T) {
	t.Parallel()
	_, err := bi.BorrowFunds(context.Background(), "", "", "", 0, 0)
	assert.ErrorIs(t, err, errLoanCoinEmpty)
	_, err = bi.BorrowFunds(context.Background(), "meow", "", "", 0, 0)
	assert.ErrorIs(t, err, errCollateralCoinEmpty)
	_, err = bi.BorrowFunds(context.Background(), "meow", "woof", "", 0, 0)
	assert.ErrorIs(t, err, errTermEmpty)
	_, err = bi.BorrowFunds(context.Background(), "meow", "woof", "neigh", 0, 0)
	assert.ErrorIs(t, err, errCollateralLoanMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.BorrowFunds(context.Background(), testCrypto.String(), testFiat.String(), "SEVEN", testAmount, 0)
	assert.NoError(t, err)
}

func TestGetOngoingLoans(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetOngoingLoans(context.Background(), 0, "", "")
	require.NoError(t, err)
	if len(resp.Data) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.GetOngoingLoans(context.Background(), resp.Data[0].OrderID, "", "")
	assert.NoError(t, err)
}

func TestGetLoanRepayHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetLoanRepayHistory(context.Background(), 0, 0, 0, "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetLoanRepayHistory(context.Background(), 0, 1, 5, "", "", time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestModifyPledgeRate(t *testing.T) {
	t.Parallel()
	_, err := bi.ModifyPledgeRate(context.Background(), 0, 0, "", "")
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = bi.ModifyPledgeRate(context.Background(), 1, 0, "", "")
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.ModifyPledgeRate(context.Background(), 1, 1, "", "")
	assert.ErrorIs(t, err, errCollateralCoinEmpty)
	_, err = bi.ModifyPledgeRate(context.Background(), 1, 1, "meow", "")
	assert.ErrorIs(t, err, errReviseTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetOngoingLoans(context.Background(), 0, "", "")
	require.NoError(t, err)
	if len(resp.Data) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.ModifyPledgeRate(context.Background(), resp.Data[0].OrderID, testAmount, testFiat.String(), "IN")
	assert.NoError(t, err)
}

func TestGetPledgeRateHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPledgeRateHistory(context.Background(), 0, 0, 0, "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetPledgeRateHistory(context.Background(), 0, 1, 5, "", "", time.Now().Add(-time.Hour*24*85),
		time.Now())
	assert.NoError(t, err)
}

func TestGetLoanHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetLoanHistory(context.Background(), 0, 0, 0, "", "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetLoanHistory(context.Background(), 0, 1, 5, "", "", "", time.Now().Add(-time.Hour*24*85), time.Now())
	assert.NoError(t, err)
}

func TestGetDebts(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// If there aren't any debts to return information on, this will return the error "The data fetched by {user ID}
	// is empty"
	testGetNoArgs(t, bi.GetDebts)
}

func TestGetLiquidationRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetLiquidationRecords(context.Background(), 0, 0, 0, "", "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetLiquidationRecords(context.Background(), 0, 1, 5, "", "", "", time.Now().Add(-time.Hour*24*85),
		time.Now())
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
	_, err := bi.UpdateTicker(context.Background(), fakePair, asset.Spot)
	assert.Error(t, err)
	_, err = bi.UpdateTicker(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.UpdateTicker(context.Background(), fakePair, asset.Futures)
	assert.Error(t, err)
	_, err = bi.UpdateTicker(context.Background(), testPair, asset.Futures)
	assert.NoError(t, err)
	_, err = bi.UpdateTicker(context.Background(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	testexch.UpdatePairsOnce(t, bi)
	err := bi.UpdateTickers(context.Background(), asset.Spot)
	assert.NoError(t, err)
	err = bi.UpdateTickers(context.Background(), asset.Futures)
	assert.NoError(t, err)
	err = bi.UpdateTickers(context.Background(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestFetchTicker(t *testing.T) {
	t.Parallel()
	_, err := bi.FetchTicker(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.FetchTicker(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
}

func TestFetchOrderbook(t *testing.T) {
	t.Parallel()
	_, err := bi.FetchOrderbook(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.FetchOrderbook(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := bi.UpdateOrderbook(context.Background(), fakePair, asset.Spot)
	assert.Error(t, err)
	_, err = bi.UpdateOrderbook(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.UpdateOrderbook(context.Background(), fakePair, asset.Futures)
	assert.Error(t, err)
	_, err = bi.UpdateOrderbook(context.Background(), testPair, asset.Futures)
	assert.NoError(t, err)
	_, err = bi.UpdateOrderbook(context.Background(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestUpdateAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := bi.UpdateAccountInfo(context.Background(), asset.Spot)
	assert.NoError(t, err)
	_, err = bi.UpdateAccountInfo(context.Background(), asset.Futures)
	assert.NoError(t, err)
	_, err = bi.UpdateAccountInfo(context.Background(), asset.Margin)
	assert.NoError(t, err)
	_, err = bi.UpdateAccountInfo(context.Background(), asset.CrossMargin)
	assert.NoError(t, err)
	_, err = bi.UpdateAccountInfo(context.Background(), asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestFetchAccountInfo(t *testing.T) {
	t.Parallel()
	_, err := bi.FetchAccountInfo(context.Background(), asset.Futures)
	assert.NoError(t, err)
	_, err = bi.FetchAccountInfo(context.Background(), asset.Futures)
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
	_, err := bi.GetWithdrawalsHistory(context.Background(), testCrypto, 0)
	assert.NoError(t, err)
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetRecentTrades(context.Background(), fakePair, asset.Spot)
	assert.Error(t, err)
	_, err = bi.GetRecentTrades(context.Background(), testPair, asset.Spot)
	assert.NoError(t, err)
	_, err = bi.GetRecentTrades(context.Background(), fakePair, asset.Futures)
	assert.Error(t, err)
	_, err = bi.GetRecentTrades(context.Background(), testPair, asset.Futures)
	assert.NoError(t, err)
	_, err = bi.GetRecentTrades(context.Background(), testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricTrades(context.Background(), fakePair, asset.Spot, time.Time{}, time.Time{})
	assert.Error(t, err)
	_, err = bi.GetHistoricTrades(context.Background(), testPair, asset.Spot, time.Now().Add(-time.Hour*24*7),
		time.Now())
	assert.NoError(t, err)
	_, err = bi.GetHistoricTrades(context.Background(), fakePair, asset.Futures, time.Time{}, time.Time{})
	assert.Error(t, err)
	_, err = bi.GetHistoricTrades(context.Background(), testPair, asset.Futures, time.Now().Add(-time.Hour*24*7),
		time.Now())
	assert.NoError(t, err)
	_, err = bi.GetHistoricTrades(context.Background(), testPair, asset.Empty, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	testGetOneArg(t, bi.GetServerTime, 0, 0, nil, false, false, true)
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	var ord *order.Submit
	_, err := bi.SubmitOrder(context.Background(), ord)
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
	_, err = bi.SubmitOrder(context.Background(), ord)
	assert.ErrorIs(t, err, errStrategyMutex)
	ord.PostOnly = false
	_, err = bi.SubmitOrder(context.Background(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ord.AssetType = asset.Futures
	_, err = bi.SubmitOrder(context.Background(), ord)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ord.AssetType = asset.Spot
	_, err = bi.SubmitOrder(context.Background(), ord)
	assert.NoError(t, err)
	ord.AssetType = asset.CrossMargin
	ord.ImmediateOrCancel = false
	ord.Side = order.Buy
	ord.Amount = testAmount2
	ord.Price = testPrice2
	_, err = bi.SubmitOrder(context.Background(), ord)
	assert.NoError(t, err)
	ord.AssetType = asset.Margin
	ord.AutoBorrow = true
	_, err = bi.SubmitOrder(context.Background(), ord)
	assert.NoError(t, err)
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := bi.GetOrderInfo(context.Background(), "", testPair, asset.Empty)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	_, err = bi.GetOrderInfo(context.Background(), "0", testPair, asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	oID := getFuturesOrdIDHelper(t, true, false)
	if oID.ClientOrderID != ordersNotFound {
		_, err = bi.GetOrderInfo(context.Background(), strconv.FormatInt(oID.OrderID, 10), testPair2, asset.Futures)
		assert.NoError(t, err)
	}
	oID = getIsoOrdIDHelper(t, false)
	if oID.ClientOrderID != ordersNotFound {
		_, err = bi.GetOrderInfo(context.Background(), strconv.FormatInt(oID.OrderID, 10), testPair2, asset.Margin)
		assert.NoError(t, err)
	}
	oID = getCrossOrdIDHelper(t, false)
	if oID.ClientOrderID != ordersNotFound {
		_, err = bi.GetOrderInfo(context.Background(), strconv.FormatInt(oID.OrderID, 10), testPair2, asset.CrossMargin)
		assert.NoError(t, err)
	}
	resp, err := bi.GetUnfilledOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62, 0)
	require.NoError(t, err)
	if len(resp.Data) != 0 {
		_, err = bi.GetOrderInfo(context.Background(), strconv.FormatInt(int64(resp.Data[0].OrderID), 10), testPair,
			asset.Spot)
		assert.NoError(t, err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := bi.GetDepositAddress(context.Background(), currency.NewCode(""), "", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.GetDepositAddress(context.Background(), testCrypto, "", "")
	assert.NoError(t, err)
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	var req *withdraw.Request
	_, err := bi.WithdrawCryptocurrencyFunds(context.Background(), req)
	assert.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil)
	req = &withdraw.Request{
		Crypto: withdraw.CryptoRequest{
			Address: testAddress,
			Chain:   testCrypto.String(),
		},
		Currency: testCrypto,
		Amount:   testAmount,
		Exchange: bi.Name,
	}
	_, err = bi.WithdrawCryptocurrencyFunds(context.Background(), req)
	assert.NoError(t, err)
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := bi.WithdrawFiatFunds(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := bi.WithdrawFiatFundsToInternationalBank(context.Background(), nil)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported)
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	var req *order.MultiOrderRequest
	_, err := bi.GetActiveOrders(context.Background(), req)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil)
	req = &order.MultiOrderRequest{
		AssetType: asset.Binary,
		Side:      order.Sell,
		Type:      order.Limit,
		Pairs:     []currency.Pair{testPair},
	}
	_, err = bi.GetActiveOrders(context.Background(), req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	req.AssetType = asset.CrossMargin
	_, err = bi.GetActiveOrders(context.Background(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Margin
	_, err = bi.GetActiveOrders(context.Background(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Futures
	_, err = bi.GetActiveOrders(context.Background(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{}
	_, err = bi.GetActiveOrders(context.Background(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Spot
	_, err = bi.GetActiveOrders(context.Background(), req)
	// This is failing since the String() method on these novel pairs returns them with a delimiter for some reason
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{testPair}
	_, err = bi.GetActiveOrders(context.Background(), req)
	assert.NoError(t, err)
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	var req *order.MultiOrderRequest
	_, err := bi.GetOrderHistory(context.Background(), req)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil)
	req = &order.MultiOrderRequest{
		AssetType: asset.Binary,
		Side:      order.Sell,
		Type:      order.Limit,
		Pairs:     []currency.Pair{testPair},
	}
	_, err = bi.GetOrderHistory(context.Background(), req)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	req.AssetType = asset.CrossMargin
	_, err = bi.GetOrderHistory(context.Background(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Margin
	_, err = bi.GetOrderHistory(context.Background(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Futures
	_, err = bi.GetOrderHistory(context.Background(), req)
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{}
	_, err = bi.GetOrderHistory(context.Background(), req)
	assert.NoError(t, err)
	req.AssetType = asset.Spot
	_, err = bi.GetOrderHistory(context.Background(), req)
	// This is failing since the String() method on these novel pairs returns them with a delimiter for some reason
	assert.NoError(t, err)
	req.Pairs = []currency.Pair{testPair}
	_, err = bi.GetOrderHistory(context.Background(), req)
	assert.NoError(t, err)
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	var fb *exchange.FeeBuilder
	_, err := bi.GetFeeByType(context.Background(), fb)
	assert.ErrorIs(t, err, common.ErrNilPointer)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	fb = &exchange.FeeBuilder{}
	_, err = bi.GetFeeByType(context.Background(), fb)
	assert.ErrorIs(t, err, errPairEmpty)
	fb.Pair = testPair
	_, err = bi.GetFeeByType(context.Background(), fb)
	assert.NoError(t, err)
	fb.IsMaker = true
	_, err = bi.GetFeeByType(context.Background(), fb)
	assert.NoError(t, err)
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	err := bi.ValidateAPICredentials(context.Background(), asset.Spot)
	assert.NoError(t, err)
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricCandles(context.Background(), currency.Pair{}, asset.Spot, kline.Raw, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	_, err = bi.GetHistoricCandles(context.Background(), testPair, asset.Spot, kline.OneDay, time.Time{}, time.Time{})
	assert.NoError(t, err)
	_, err = bi.GetHistoricCandles(context.Background(), testPair, asset.Futures, kline.OneDay, time.Time{}, time.Time{})
	assert.NoError(t, err)

	// _, err = bi.GetHistoricCandles(context.Background(), testPair, asset.Binary, kline.OneMin, time.Now().Add(-time.Hour),
	// 	time.Now())
	// assert.ErrorIs(t, err, asset.ErrNotSupported)
	// _, err = bi.GetHistoricCandles(context.Background(), testPair, asset.Spot, kline.OneMin, time.Now().Add(-time.Hour),
	// time.Now())
	// assert.NoError(t, err)
	// _, err = bi.GetHistoricCandles(context.Background(), testPair, asset.Futures, kline.OneMin, time.Now().Add(-time.Hour),
	// 	time.Now())
	// assert.NoError(t, err)
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricCandlesExtended(context.Background(), currency.Pair{}, asset.Spot, kline.Raw, time.Time{},
		time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty)
	// Rest of this is being put on ice until the issue with the previous test has been figured out
}

// The following 3 tests aren't parallel due to collisions with each other, and some other plan order-related tests
func TestModifyPlanSpotOrder(t *testing.T) {
	_, err := bi.ModifyPlanSpotOrder(context.Background(), 0, "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.ModifyPlanSpotOrder(context.Background(), 0, "meow", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.ModifyPlanSpotOrder(context.Background(), 0, "meow", "woof", 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.ModifyPlanSpotOrder(context.Background(), 0, "meow", "limit", 1, 0, 0)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = bi.ModifyPlanSpotOrder(context.Background(), 0, "meow", "woof", 1, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordID, err := bi.GetCurrentSpotPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	if len(ordID.Data.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := bi.ModifyPlanSpotOrder(context.Background(), ordID.Data.OrderList[0].OrderID,
		ordID.Data.OrderList[0].ClientOrderID, "limit", testPrice, testPrice, testAmount)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestCancelPlanSpotOrder(t *testing.T) {
	_, err := bi.CancelPlanSpotOrder(context.Background(), 0, "")
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordID, err := bi.GetCurrentSpotPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	require.NotNil(t, ordID)
	if len(ordID.Data.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := bi.CancelPlanSpotOrder(context.Background(), ordID.Data.OrderList[0].OrderID,
		ordID.Data.OrderList[0].ClientOrderID)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestModifyOrder(t *testing.T) {
	var ord *order.Modify
	_, err := bi.ModifyOrder(context.Background(), ord)
	assert.ErrorIs(t, err, order.ErrModifyOrderIsNil)
	ord = &order.Modify{
		Pair:      testPair,
		AssetType: 1<<31 - 1,
		OrderID:   "meow",
	}
	_, err = bi.ModifyOrder(context.Background(), ord)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	ord.OrderID = "0"
	_, err = bi.ModifyOrder(context.Background(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ord.AssetType = asset.Futures
	_, err = bi.ModifyOrder(context.Background(), ord)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordID, err := bi.GetCurrentSpotPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	if len(ordID.Data.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	ord.OrderID = strconv.FormatInt(ordID.Data.OrderList[0].OrderID, 10)
	ord.ClientOrderID = ordID.Data.OrderList[0].ClientOrderID
	ord.Type = order.Limit
	ord.Price = testPrice
	ord.TriggerPrice = testPrice
	ord.Amount = testAmount
	ord.AssetType = asset.Spot
	_, err = bi.ModifyOrder(context.Background(), ord)
	assert.NoError(t, err)
}

func TestCommitConversion(t *testing.T) {
	// In a separate parallel batch due to collision with TestGetQuotedPrice
	t.Parallel()
	_, err := bi.CommitConversion(context.Background(), "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.CommitConversion(context.Background(), testCrypto.String(), testFiat.String(), "", 0, 0, 0)
	assert.ErrorIs(t, err, errTraceIDEmpty)
	_, err = bi.CommitConversion(context.Background(), testCrypto.String(), testFiat.String(), "1", 0, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.CommitConversion(context.Background(), testCrypto.String(), testFiat.String(), "1", 1, 1, 0)
	assert.ErrorIs(t, err, errPriceEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), testAmount, 0)
	require.NoError(t, err)
	_, err = bi.CommitConversion(context.Background(), testCrypto.String(), testFiat.String(), resp.Data.TraceID,
		resp.Data.FromCoinSize, resp.Data.ToCoinSize, resp.Data.ConvertPrice)
	assert.NoError(t, err)
}

func TestCancelTriggerFuturesOrders(t *testing.T) {
	// In a separate parallel batch due to collisions with TestModifyTPSLFuturesOrder and TestModifyTriggerFuturesOrder
	t.Parallel()
	var ordList []OrderIDStruct
	_, err := bi.CancelTriggerFuturesOrders(context.Background(), ordList, "", "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	oID := getTrigOrdIDHelper(t, []string{"profit_loss", "normal_plan", "track_plan"})
	ordList = append(ordList, *oID)
	resp, err := bi.CancelTriggerFuturesOrders(context.Background(), ordList, testPair2.String(),
		testFiat2.String()+"-FUTURES", "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestRepayLoan(t *testing.T) {
	// In a separate parallel batch due to a collision with ModifyPledgeRate
	t.Parallel()
	_, err := bi.RepayLoan(context.Background(), 0, 0, false, false)
	assert.ErrorIs(t, err, errOrderIDEmpty)
	_, err = bi.RepayLoan(context.Background(), 1, 0, false, false)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetOngoingLoans(context.Background(), 0, "", "")
	require.NoError(t, err)
	if len(resp.Data) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.RepayLoan(context.Background(), resp.Data[0].OrderID, testAmount, false, false)
	assert.NoError(t, err)
	_, err = bi.RepayLoan(context.Background(), resp.Data[0].OrderID, 0, true, true)
	assert.NoError(t, err)
}

// The following 7 tests aren't parallel due to collisions with each other, and some other futures-related tests
func TestModifyFuturesOrder(t *testing.T) {
	_, err := bi.ModifyFuturesOrder(context.Background(), 0, "", "", "", "", 0, 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.ModifyFuturesOrder(context.Background(), 1, "", "", "", "", 0, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.ModifyFuturesOrder(context.Background(), 1, "", "meow", "", "", 0, 0, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ModifyFuturesOrder(context.Background(), 1, "", "meow", "woof", "", 0, 0, 0, 0)
	assert.ErrorIs(t, err, errNewClientOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	oID := getFuturesOrdIDHelper(t, true, true)
	_, err = bi.ModifyFuturesOrder(context.Background(), oID.OrderID, oID.ClientOrderID, testPair2.String(),
		testFiat2.String()+"-FUTURES", clientIDGenerator(), 0, 0, testPrice2+1, testPrice2/10)
	assert.NoError(t, err)
}

func TestCancelFuturesOrder(t *testing.T) {
	_, err := bi.CancelFuturesOrder(context.Background(), "", "", "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelFuturesOrder(context.Background(), "meow", "", "", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.CancelFuturesOrder(context.Background(), "meow", "woof", "", "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	oID := getFuturesOrdIDHelper(t, true, true)
	_, err = bi.CancelFuturesOrder(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES", "",
		"", oID.OrderID)
	assert.NoError(t, err)
}

func TestBatchCancelFuturesOrders(t *testing.T) {
	_, err := bi.BatchCancelFuturesOrders(context.Background(), nil, "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	oID := getFuturesOrdIDHelper(t, true, true)
	orders := []OrderIDStruct{
		{
			OrderID: oID.OrderID,
		},
	}
	_, err = bi.BatchCancelFuturesOrders(context.Background(), orders, testPair2.String(),
		testFiat2.String()+"-FUTURES", "")
	assert.NoError(t, err)
}

func TestFlashClosePosition(t *testing.T) {
	_, err := bi.FlashClosePosition(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.FlashClosePosition(context.Background(), testPair2.String(), "", testFiat2.String()+"-FUTURES")
	assert.NoError(t, err)
}

func TestCancelAllFuturesOrders(t *testing.T) {
	_, err := bi.CancelAllFuturesOrders(context.Background(), "", "", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CancelAllFuturesOrders(context.Background(), "", testFiat2.String()+"-FUTURES", testFiat2.String(),
		time.Second*60)
	assert.NoError(t, err)
}

func TestCancelOrder(t *testing.T) {
	var ord *order.Cancel
	err := bi.CancelOrder(context.Background(), ord)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil)
	ord = &order.Cancel{
		OrderID:   "meow",
		AssetType: 1<<31 - 1,
	}
	err = bi.CancelOrder(context.Background(), ord)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	ord.OrderID = "0"
	err = bi.CancelOrder(context.Background(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	ord.AssetType = asset.Margin
	err = bi.CancelOrder(context.Background(), ord)
	assert.ErrorIs(t, err, errPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.GetUnfilledOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62, 0)
	require.NoError(t, err)
	if len(resp.Data) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	ord.OrderID = strconv.FormatInt(int64(resp.Data[0].OrderID), 10)
	ord.Pair = testPair
	ord.AssetType = asset.Spot
	ord.ClientOrderID = resp.Data[0].ClientOrderID
	err = bi.CancelOrder(context.Background(), ord)
	assert.NoError(t, err)
	oID := getFuturesOrdIDHelper(t, true, true)
	ord.OrderID = strconv.FormatInt(oID.OrderID, 10)
	ord.Pair = testPair2
	ord.AssetType = asset.Futures
	ord.ClientOrderID = oID.ClientOrderID
	err = bi.CancelOrder(context.Background(), ord)
	assert.NoError(t, err)
	oID = getIsoOrdIDHelper(t, true)
	ord.OrderID = strconv.FormatInt(oID.OrderID, 10)
	ord.AssetType = asset.CrossMargin
	ord.ClientOrderID = oID.ClientOrderID
	err = bi.CancelOrder(context.Background(), ord)
	assert.NoError(t, err)
}

func TestCancelAllOrders(t *testing.T) {
	var ord *order.Cancel
	_, err := bi.CancelAllOrders(context.Background(), ord)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil)
	ord = &order.Cancel{
		AssetType: asset.Empty,
	}
	_, err = bi.CancelAllOrders(context.Background(), ord)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ord.AssetType = asset.Spot
	ord.Pair = testPair
	_, err = bi.CancelAllOrders(context.Background(), ord)
	assert.NoError(t, err)
	ord.AssetType = asset.Futures
	ord.Pair = testPair2
	_, err = bi.CancelAllOrders(context.Background(), ord)
	assert.NoError(t, err)
}

// The following 3 tests aren't parallel due to collisions with each other, and some other cross-related tests
func TestCancelCrossOrder(t *testing.T) {
	_, err := bi.CancelCrossOrder(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelCrossOrder(context.Background(), "meow", "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	oID := getCrossOrdIDHelper(t, true)
	_, err = bi.CancelCrossOrder(context.Background(), testPair.String(), oID.ClientOrderID, oID.OrderID)
	assert.NoError(t, err)
}

func TestBatchCancelCrossOrders(t *testing.T) {
	_, err := bi.BatchCancelCrossOrders(context.Background(), "", nil)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchCancelCrossOrders(context.Background(), "meow", nil)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	oID := getCrossOrdIDHelper(t, true)
	_, err = bi.BatchCancelCrossOrders(context.Background(), testPair.String(), []OrderIDStruct{*oID})
	assert.NoError(t, err)
}

func TestCancelBatchOrders(t *testing.T) {
	var orders []order.Cancel
	orders = append(orders, order.Cancel{
		AssetType: asset.Empty,
	})
	_, err := bi.CancelBatchOrders(context.Background(), orders)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	orders[0].OrderID = "0"
	_, err = bi.CancelBatchOrders(context.Background(), orders)
	assert.ErrorIs(t, err, asset.ErrNotSupported)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	orders = nil
	resp, err := bi.GetUnfilledOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62, 0)
	require.NoError(t, err)
	if len(resp.Data) != 0 {
		orders = append(orders, order.Cancel{
			AssetType:     asset.Spot,
			OrderID:       strconv.FormatInt(int64(resp.Data[0].OrderID), 10),
			ClientOrderID: resp.Data[0].ClientOrderID,
			Pair:          testPair,
		})
	}
	oID := getFuturesOrdIDHelper(t, true, false)
	if oID.ClientOrderID != ordersNotFound {
		orders = append(orders, order.Cancel{
			AssetType:     asset.Futures,
			OrderID:       strconv.FormatInt(oID.OrderID, 10),
			ClientOrderID: oID.ClientOrderID,
			Pair:          testPair2,
		})
	}
	oID = getIsoOrdIDHelper(t, false)
	if oID.ClientOrderID != ordersNotFound {
		orders = append(orders, order.Cancel{
			AssetType:     asset.Margin,
			OrderID:       strconv.FormatInt(oID.OrderID, 10),
			ClientOrderID: oID.ClientOrderID,
			Pair:          testPair2,
		})
	}
	oID = getCrossOrdIDHelper(t, false)
	if oID.ClientOrderID != ordersNotFound {
		orders = append(orders, order.Cancel{
			AssetType:     asset.CrossMargin,
			OrderID:       strconv.FormatInt(oID.OrderID, 10),
			ClientOrderID: oID.ClientOrderID,
			Pair:          testPair2,
		})
	}
	if len(orders) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	_, err = bi.CancelBatchOrders(context.Background(), orders)
	assert.NoError(t, err)
}

// The following 2 tests aren't parallel due to collisions with each other, and some other isolated-related tests
func TestCancelIsolatedOrder(t *testing.T) {
	_, err := bi.CancelIsolatedOrder(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelIsolatedOrder(context.Background(), "meow", "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	oID := getIsoOrdIDHelper(t, true)
	_, err = bi.CancelIsolatedOrder(context.Background(), testPair2.String(), oID.ClientOrderID, oID.OrderID)
	assert.NoError(t, err)
}

func TestBatchCancelIsolatedOrders(t *testing.T) {
	_, err := bi.BatchCancelIsolatedOrders(context.Background(), "", nil)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchCancelIsolatedOrders(context.Background(), "meow", nil)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	oID := getIsoOrdIDHelper(t, true)
	_, err = bi.BatchCancelIsolatedOrders(context.Background(), testPair2.String(), []OrderIDStruct{*oID})
	assert.NoError(t, err)
}

type getNoArgsResp interface {
	*TimeResp | *P2PMerInfoResp | *ConvertCoinsResp | *BGBConvertCoinsResp | *VIPFeeRateResp | *SupCurrencyResp |
		*RiskRateCross | *SavingsBalance | *SharkFinBalance | *DebtsResp | *AssetOverviewResp | *BGBDeductResp |
		*SymbolsResp | *SubaccountAssetsResp | []exchange.FundingHistory
}

type getNoArgsAssertNotEmpty[G getNoArgsResp] func(context.Context) (G, error)

func testGetNoArgs[G getNoArgsResp](t *testing.T, f getNoArgsAssertNotEmpty[G]) {
	t.Helper()
	_, err := f(context.Background())
	assert.NoError(t, err)
}

type getOneArgResp interface {
	*WhaleNetFlowResp | *FundFlowResp | *WhaleFundFlowResp | *CrVirSubResp | *GetAPIKeyResp | *FundingAssetsResp |
		*BotAccAssetsResp | *ConvertBGBResp | *CoinInfoResp | *SymbolInfoResp | *TickerResp | *SymbolResp |
		*SubOrderResp | *BatchOrderResp | *BoolData | *FutureTickerResp | *AllAccResp | *SubaccountFuturesResp |
		*CrossAssetResp | *MaxBorrowCross | *MaxTransferCross | *IntRateMaxBorrowCross | *TierConfigCross |
		*FlashRepayCross | *IsoAssetResp | *IntRateMaxBorrowIso | *MaxBorrowIso | *MaxTransferIso | *FlashRepayIso |
		*EarnAssets | *LoanCurList | currency.Pairs | time.Time
}

type getOneArgParam interface {
	string | []string | bool | asset.Item
}

type getOneArgGen[R getOneArgResp, P getOneArgParam] func(context.Context, P) (R, error)

func testGetOneArg[R getOneArgResp, P getOneArgParam](t *testing.T, f getOneArgGen[R, P], callErrCheck, callNoEErr P, tarErr error, checkResp, checkCreds, canManipOrders bool) {
	t.Helper()
	if tarErr != nil {
		_, err := f(context.Background(), callErrCheck)
		assert.ErrorIs(t, err, tarErr)
	}
	if checkCreds {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipOrders)
	}
	resp, err := f(context.Background(), callNoEErr)
	require.NoError(t, err)
	if checkResp {
		assert.NotEmpty(t, resp)
	}
}

type getTwoArgsResp interface {
	*FutureTickerResp | *OpenPositionsResp | *FundingTimeResp | *FuturesPriceResp | *FundingCurrentResp |
		*ContractConfigResp
}

type getTwoArgsPairProduct[G getTwoArgsResp] func(context.Context, string, string) (G, error)

func testGetTwoArgs[G getTwoArgsResp](t *testing.T, f getTwoArgsPairProduct[G]) {
	t.Helper()
	_, err := f(context.Background(), "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = f(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = f(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES")
	assert.NoError(t, err)
}

func subAccTestHelper(t *testing.T, compString, ignoreString string) string {
	t.Helper()
	resp, err := bi.GetVirtualSubaccounts(context.Background(), 25, 0, "")
	assert.NoError(t, err)
	require.NotEmpty(t, resp)
	tarID := ""
	for i := range resp.Data.SubaccountList {
		if resp.Data.SubaccountList[i].SubaccountName == compString &&
			resp.Data.SubaccountList[i].SubaccountName != ignoreString {
			tarID = resp.Data.SubaccountList[i].SubaccountUID
			break
		}
		if compString == "" && resp.Data.SubaccountList[i].SubaccountName != ignoreString {
			tarID = resp.Data.SubaccountList[i].SubaccountUID
			break
		}
	}
	if tarID == "" {
		t.Skipf(skipTestSubAccNotFound, compString, ignoreString)
	}
	return tarID
}

func getPlanOrdIDHelper(t *testing.T, mustBeTriggered bool) *OrderIDStruct {
	t.Helper()
	ordIDs := new(OrderIDStruct)
	resp, err := bi.GetCurrentSpotPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 100,
		1<<62)
	if err == nil && len(resp.Data.OrderList) != 0 {
		for i := range resp.Data.OrderList {
			if resp.Data.OrderList[i].ClientOrderID == url.QueryEscape(resp.Data.OrderList[i].ClientOrderID) &&
				!(mustBeTriggered && resp.Data.OrderList[i].Status == "not_trigger") {
				ordIDs.ClientOrderID = resp.Data.OrderList[i].ClientOrderID
				ordIDs.OrderID = resp.Data.OrderList[i].OrderID
			}
		}
	}
	if ordIDs.ClientOrderID == "" {
		t.Skip(skipInsufficientOrders)
	}
	return ordIDs
}

func getFuturesOrdIDHelper(t *testing.T, live, skip bool) *OrderIDStruct {
	t.Helper()
	resp, err := bi.GetPendingFuturesOrders(context.Background(), 0, 1<<62, 5, "", testPair2.String(),
		testFiat2.String()+"-FUTURES", "", time.Now().Add(-time.Hour*24*90), time.Now())
	assert.NoError(t, err)
	if resp != nil {
		if len(resp.Data.EntrustedList) != 0 {
			return &OrderIDStruct{
				OrderID:       resp.Data.EntrustedList[0].OrderID,
				ClientOrderID: resp.Data.EntrustedList[0].ClientOrderID,
			}
		}
	}
	if !live {
		resp, err := bi.GetHistoricalFuturesOrders(context.Background(), 0, 1<<62, 5, "", testPair2.String(),
			testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
		assert.NoError(t, err)
		if resp != nil {
			if len(resp.Data.EntrustedList) != 0 {
				return &OrderIDStruct{
					OrderID:       resp.Data.EntrustedList[0].OrderID,
					ClientOrderID: resp.Data.EntrustedList[0].ClientOrderID,
				}
			}
		}
	}
	if skip {
		t.Skip(skipInsufficientOrders)
	}
	return &OrderIDStruct{
		OrderID:       0,
		ClientOrderID: ordersNotFound,
	}
}

func getTrigOrdIDHelper(t *testing.T, planTypes []string) *OrderIDStruct {
	t.Helper()
	for i := range planTypes {
		resp, err := bi.GetPendingTriggerFuturesOrders(context.Background(), 0, 1<<62, 5, "", testPair2.String(),
			planTypes[i], testFiat2.String()+"-FUTURES", time.Time{}, time.Time{})
		assert.NoError(t, err)
		if resp != nil {
			if len(resp.Data.EntrustedList) != 0 {
				return &OrderIDStruct{
					OrderID:       resp.Data.EntrustedList[0].OrderID,
					ClientOrderID: resp.Data.EntrustedList[0].ClientOrderID,
				}
			}
		}
	}
	t.Skip(skipInsufficientOrders)
	return &OrderIDStruct{
		OrderID:       0,
		ClientOrderID: ordersNotFound,
	}
}

func getCrossOrdIDHelper(t *testing.T, skip bool) *OrderIDStruct {
	t.Helper()
	resp, err := bi.GetCrossOpenOrders(context.Background(), testPair.String(), "", 0, 5, 1<<62,
		time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Data.OrderList) != 0 {
		return &OrderIDStruct{
			OrderID:       resp.Data.OrderList[0].OrderID,
			ClientOrderID: resp.Data.OrderList[0].ClientOrderID,
		}
	}
	if skip {
		t.Skip(skipInsufficientOrders)
	}
	return &OrderIDStruct{
		OrderID:       0,
		ClientOrderID: ordersNotFound,
	}
}

func getIsoOrdIDHelper(t *testing.T, skip bool) *OrderIDStruct {
	t.Helper()
	resp, err := bi.GetIsolatedOpenOrders(context.Background(), testPair.String(), "", 0, 5, 1<<62,
		time.Now().Add(-time.Hour*24*85), time.Time{})
	require.NoError(t, err)
	if len(resp.Data.OrderList) != 0 {
		return &OrderIDStruct{
			OrderID:       resp.Data.OrderList[0].OrderID,
			ClientOrderID: resp.Data.OrderList[0].ClientOrderID,
		}
	}
	if skip {
		t.Skip(skipInsufficientOrders)
	}
	return &OrderIDStruct{
		OrderID:       0,
		ClientOrderID: ordersNotFound,
	}
}

func aBenchmarkHelper(a, pag int64) {
	var params Params
	params.Values = make(url.Values)
	params.Values.Set("limit", strconv.FormatInt(a, 10))
	params.Values.Set("idLessThan", strconv.FormatInt(pag, 10))
}

// irrelevant/outdated data retained for formatting
// 4952	    292175 ns/op	  165455 B/op	       4 allocs/op
func BenchmarkGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var sizableArray [][10]int
		var done bool
		for !done {
			// tempArray := make([][10]int, 100)
			// for x := 0; x < 100; x++ {
			// 	tempArray[x] = [10]int{5, 1 << 30, i % 27, x % 9, x ^ i, 2, 3, 4, 5, 6}
			// }
			// sizableArray = append(sizableArray, tempArray...)
			for x := 0; x < 100; x++ {
				sizableArray = append(sizableArray, [10]int{5, 1 << 30, i % 27, x % 9, x ^ i, 2, 3, 4, 5, 6})
			}
			if i%5 == 0 || len(sizableArray) > 1000 {
				done = true
			}
		}
	}
}
