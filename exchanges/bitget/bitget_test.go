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
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
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
	// Test values used with live data, with the goal of never letting an order be executed
	testAmount = 0.001
	testPrice  = 1e10 - 1
	// Test values used with demo functionality, with the goal of lining up with the relatively strict currency
	// limits present there
	testAmount2 = 0.003
	testPrice2  = 0.1
	testAddress = "fake test address"
)

// User-defined variables to aid testing
var (
	testCrypto  = currency.BTC   // Used for endpoints which don't support demo trading
	testCrpyo2  = currency.SBTC  // Used for endpoints which support demo trading
	testCrypto3 = currency.DOGE  // Used for endpoints which consume all available funds
	testFiat    = currency.USDT  // Used for endpoints which don't support demo trading
	testFiat2   = currency.SUSDT // Used for endpoints which support demo trading
	testPair    = currency.NewPair(testCrypto, testFiat)
	testPair2   = currency.NewPair(testCrpyo2, testFiat2)
)

// Developer-defined constants to aid testing
const (
	skipTestSubAccNotFound       = "appropriate sub-account (equals %v, not equals %v) not found, skipping"
	skipInsufficientAPIKeysFound = "insufficient API keys found, skipping"
	skipInsufficientBalance      = "insufficient balance to place order, skipping"
	skipInsufficientOrders       = "insufficient orders found, skipping"

	errAPIKeyLimitPartial              = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"40063","msg":"API exceeds the maximum limit added","requestTime":`
	errInsufficientBalancePartial      = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"43012","msg":"Insufficient balance","requestTime":`
	errAmountExceedsBalancePartial     = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"40762","msg":"The order amount exceeds the balance","requestTime":`
	errCurrentlyHoldingPositionPartial = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"45117","msg":"Currently holding positions or orders, the margin mode cannot be adjusted","requestTime":`
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
	_, err := bi.QueryAnnouncements(context.Background(), "", time.Now().Add(time.Hour), time.Now())
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	resp, err := bi.QueryAnnouncements(context.Background(), "latest_news", time.Time{}, time.Time{})
	assert.NoError(t, err)
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
	_, err = bi.GetTradeRate(context.Background(), "BTCUSDT", "")
	assert.ErrorIs(t, err, errBusinessTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetTradeRate(context.Background(), "BTCUSDT", "spot")
	assert.NoError(t, err)
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
	// Can't currently be properly tested due to not knowing any p2p merchant IDs
	_, err := bi.GetP2PMerchantList(context.Background(), "", "1", 5, 1<<62)
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

func TestCreateVirtualSubaccounts(t *testing.T) {
	t.Parallel()
	_, err := bi.CreateVirtualSubaccounts(context.Background(), []string{})
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.CreateVirtualSubaccounts(context.Background(), []string{testSubaccountName})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
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
	assert.NoError(t, err)
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
	_, err := bi.GetVirtualSubaccounts(context.Background(), 25, 1<<62, "")
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
	_, err := bi.GetAPIKeys(context.Background(), "")
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := subAccTestHelper(t, strings.ToLower(string(testSubaccountName[:3]))+"****@virtual-bitget.com", "")
	resp, err := bi.GetAPIKeys(context.Background(), tarID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
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
	resp, err := bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), 0, 1)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
	resp, err = bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), 0.1, 0)
	assert.NoError(t, err)
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
	var currencies []string
	_, err := bi.ConvertBGB(context.Background(), currencies)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	currencies = append(currencies, testCrypto3.String())
	// No matter what currency I use, this returns the error "currency does not support convert"; possibly a bad error
	// message, with the true issue being lack of funds?
	_, err = bi.ConvertBGB(context.Background(), currencies)
	assert.NoError(t, err)
}

func TestGetBGBConvertHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetBGBConvertHistory(context.Background(), 0, 0, 0, time.Now(), time.Now().Add(-time.Second))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetBGBConvertHistory(context.Background(), 0, 5, 1<<62, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetCoinInfo(t *testing.T) {
	t.Parallel()
	resp, err := bi.GetCoinInfo(context.Background(), "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSymbolInfo(t *testing.T) {
	t.Parallel()
	resp, err := bi.GetSymbolInfo(context.Background(), "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSpotVIPFeeRate(t *testing.T) {
	t.Parallel()
	testGetNoArgs(t, bi.GetSpotVIPFeeRate)
}

func TestGetSpotTickerInformation(t *testing.T) {
	t.Parallel()
	resp, err := bi.GetSpotTickerInformation(context.Background(), testPair.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSpotMergeDepth(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotMergeDepth(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetSpotMergeDepth(context.Background(), testPair.String(), "scale3", "5")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetOrderbookDepth(t *testing.T) {
	t.Parallel()
	resp, err := bi.GetOrderbookDepth(context.Background(), testPair.String(), "", 5)
	assert.NoError(t, err)
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
	_, err = bi.GetSpotCandlestickData(context.Background(), "meow", "woof", time.Now(), time.Now().Add(-time.Second),
		0, false)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	resp, err := bi.GetSpotCandlestickData(context.Background(), testPair.String(), "1min", time.Time{}, time.Time{},
		5, false)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.SpotCandles)
	resp, err = bi.GetSpotCandlestickData(context.Background(), testPair.String(), "1min", time.Time{}, time.Now(), 5,
		true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.SpotCandles)
}

func TestGetRecentSpotFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetRecentSpotFills(context.Background(), "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetRecentSpotFills(context.Background(), testPair.String(), 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSpotMarketTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotMarketTrades(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotMarketTrades(context.Background(), "meow", time.Now(), time.Now().Add(-time.Second), 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	resp, err := bi.GetSpotMarketTrades(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
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
	_, err = bi.CancelSpotOrderByID(context.Background(), "meow", "a", -1)
	if err == nil {
		t.Error(errNonsenseRequest)
	}
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
	assert.NoError(t, err)
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
	_, err = bi.BatchCancelOrders(context.Background(), "meow", []OrderIDStruct{{}})
	if err == nil {
		t.Error(errNonsenseRequest)
	}
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
	_, err := bi.CancelOrderBySymbol(context.Background(), "")
	assert.ErrorIs(t, err, errPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.CancelOrderBySymbol(context.Background(), testPair.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
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
	_, err := bi.GetUnfilledOrders(context.Background(), "", time.Now(), time.Now().Add(-time.Second), 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetUnfilledOrders(context.Background(), "", time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
}

func TestGetHistoricalOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalOrders(context.Background(), "", time.Now(), time.Now().Add(-time.Second), 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetHistoricalOrders(context.Background(), "", time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
}

func TestGetSpotFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSpotFills(context.Background(), "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotFills(context.Background(), "meow", time.Now(), time.Now().Add(-time.Second), 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotFills(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
}

func TestPlacePlanOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlacePlanOrder(context.Background(), "", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlacePlanOrder(context.Background(), "meow", "", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlacePlanOrder(context.Background(), "meow", "woof", "", "", "", "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.PlacePlanOrder(context.Background(), "meow", "woof", "", "", "", "", "", 1, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlacePlanOrder(context.Background(), "meow", "woof", "limit", "", "", "", "", 1, 0, 0)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = bi.PlacePlanOrder(context.Background(), "meow", "woof", "neigh", "", "", "", "", 1, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.PlacePlanOrder(context.Background(), "meow", "woof", "neigh", "", "", "", "", 1, 0, 1)
	assert.ErrorIs(t, err, errTriggerTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	cID := clientIDGenerator()
	resp, err := bi.PlacePlanOrder(context.Background(), testPair.String(), "sell", "limit", "", "fill_price", cID,
		"ioc", testPrice, testPrice, testAmount)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestModifyPlanOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.ModifyPlanOrder(context.Background(), 0, "", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	_, err = bi.ModifyPlanOrder(context.Background(), 0, "meow", "", 0, 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.ModifyPlanOrder(context.Background(), 0, "meow", "woof", 0, 0, 0)
	assert.ErrorIs(t, err, errTriggerPriceEmpty)
	_, err = bi.ModifyPlanOrder(context.Background(), 0, "meow", "limit", 1, 0, 0)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = bi.ModifyPlanOrder(context.Background(), 0, "meow", "woof", 1, 0, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordID, err := bi.GetCurrentPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	if len(ordID.Data.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := bi.ModifyPlanOrder(context.Background(), ordID.Data.OrderList[0].OrderID,
		ordID.Data.OrderList[0].ClientOrderID, "limit", testPrice, testPrice, testAmount)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestCancelPlanOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.CancelPlanOrder(context.Background(), 0, "")
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordID, err := bi.GetCurrentPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	if len(ordID.Data.OrderList) == 0 {
		t.Skip(skipInsufficientOrders)
	}
	resp, err := bi.CancelPlanOrder(context.Background(), ordID.Data.OrderList[0].OrderID,
		ordID.Data.OrderList[0].ClientOrderID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetCurrentPlanOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetCurrentPlanOrders(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCurrentPlanOrders(context.Background(), "meow", time.Now(), time.Now().Add(-time.Second), 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetCurrentPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetPlanSubOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPlanSubOrder(context.Background(), "")
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	ordIDs := getPlanOrdIDHelper(t, true)
	resp, err := bi.GetPlanSubOrder(context.Background(), strconv.FormatInt(ordIDs.OrderID, 10))
	// This gets the error "the current plan order does not exist or has not been triggered" even when using
	// a plan order that definitely exists and has definitely been triggered
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetPlanOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPlanOrderHistory(context.Background(), "", time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetPlanOrderHistory(context.Background(), "meow", time.Time{}, time.Time{}, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetPlanOrderHistory(context.Background(), testPair.String(), time.Now().Add(-time.Hour*24*90),
		time.Now().Add(-time.Minute), 1<<30)
	assert.NoError(t, err)
}

func TestBatchCancelPlanOrders(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err := bi.BatchCancelPlanOrders(context.Background(), []string{testPair.String()})
	assert.NoError(t, err)
}

func TestGetAccountInfo(t *testing.T) {
	t.Parallel()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetAccountInfo(context.Background())
	assert.NoError(t, err)
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
	_, err := bi.GetSpotSubaccountAssets(context.Background())
	assert.NoError(t, err)
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
	_, err := bi.GetSpotAccountBills(context.Background(), "", "", "", time.Now(), time.Now().Add(-time.Minute), 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
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
	cID := clientIDGenerator()
	_, err = bi.TransferAsset(context.Background(), "spot", "p2p", testCrypto.String(), testPair.String(), cID,
		testAmount)
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
	assert.NoError(t, err)
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
	cID := clientIDGenerator()
	_, err = bi.SubaccountTransfer(context.Background(), "spot", "p2p", testCrypto.String(), testPair.String(), cID,
		fromID, toID, testAmount)
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
	cID := clientIDGenerator()
	_, err = bi.WithdrawFunds(context.Background(), testCrypto.String(), "on_chain", testAddress, "", "", "", "",
		"", cID, testAmount)
	assert.NoError(t, err)
}

func TestGetSubaccountTransferRecord(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSubaccountTransferRecord(context.Background(), "", "", "", time.Now(), time.Now().Add(-time.Minute),
		0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
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
	_, err = bi.GetTransferRecord(context.Background(), "meow", "woof", "", time.Now(), time.Now().Add(-time.Minute), 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetTransferRecord(context.Background(), testCrypto.String(), "spot", "meow", time.Time{}, time.Time{},
		3, 1<<62)
	assert.NoError(t, err)
}

func TestGetDepositAddressForCurrency(t *testing.T) {
	t.Parallel()
	_, err := bi.GetDepositAddressForCurrency(context.Background(), "", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetDepositAddressForCurrency(context.Background(), testCrypto.String(), "")
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetSubaccountDepositRecords(t *testing.T) {
	t.Parallel()
	_, err := bi.GetSubaccountDepositRecords(context.Background(), "", "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.GetSubaccountDepositRecords(context.Background(), "meow", "", 0, 0, 0, time.Now(),
		time.Now().Add(-time.Minute))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := subAccTestHelper(t, "", "")
	_, err = bi.GetSubaccountDepositRecords(context.Background(), tarID, "", 0, 1<<62, 2, time.Time{}, time.Time{})
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
	assert.NoError(t, err)
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
	_, err := bi.GetAllFuturesTickers(context.Background(), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetAllFuturesTickers(context.Background(), "COIN-FUTURES")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetRecentFuturesFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetRecentFuturesFills(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetRecentFuturesFills(context.Background(), "meow", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetRecentFuturesFills(context.Background(), testPair.String(), "USDT-FUTURES", 5)
	assert.NoError(t, err)
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
	_, err = bi.GetFuturesCandlestickData(context.Background(), "meow", "woof", "neigh", time.Now(),
		time.Now().Add(-time.Minute), 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetAllFuturesAccounts(t *testing.T) {
	t.Parallel()
	_, err := bi.GetAllFuturesAccounts(context.Background(), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetAllFuturesAccounts(context.Background(), "COIN-FUTURES")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetFuturesSubaccountAssets(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesSubaccountAssets(context.Background(), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetFuturesSubaccountAssets(context.Background(), "USDT-FUTURES")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
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
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Data)
}

func TestGetFuturesAccountBills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesAccountBills(context.Background(), "", "", "", "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesAccountBills(context.Background(), "meow", "", "", "", 0, 0, time.Now(),
		time.Now().Add(-time.Minute))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
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
	assert.NoError(t, err)
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
	_, err := bi.GetHistoricalPositions(context.Background(), "", "", 0, 0, time.Now(),
		time.Now().Add(-time.Minute))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetHistoricalPositions(context.Background(), "", "", 2>>62, 5, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestPlaceFuturesOrder(t *testing.T) {
	t.Parallel()
	_, err := bi.PlaceFuturesOrder(context.Background(), "", "", "", "", "", "", "", "", "", "", "", 0, 0, false,
		false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "", "", "", "", "", "", "", "", "", "", 0, 0, false,
		false)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "", "", "", "", "", "", "", "", "", 0, 0,
		false, false)
	assert.ErrorIs(t, err, errMarginModeEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "", "", "", "", "", "", "", "", 0,
		0, false, false)
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "", "", "", "", "", "", "",
		0, 0, false, false)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "", "", "", "",
		"", "", 0, 0, false, false)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "", "limit", "",
		"", "", "", 0, 0, false, false)
	assert.ErrorIs(t, err, errAmountEmpty)
	_, err = bi.PlaceFuturesOrder(context.Background(), "meow", "woof", "neigh", "oink", "quack", "", "limit", "",
		"", "", "", 1, 0, false, false)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	resp := placeFuturesOrderHelper(t)
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
	cID := clientIDGenerator()
	// Once you can get all the pending futures orders, check those for any that can be reversed. If none are
	// found, place one to be reversed. If that placement fails due to insufficient balance, skip.
	_, err = bi.PlaceReversal(context.Background(), testPair2.String(), testFiat2.String(),
		testFiat2.String()+"-FUTURES", "close_long", "Buy", cID, testAmount, true)
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
	oID := getFuturesOrdIDHelper(t)
	_, err = bi.GetFuturesOrderDetails(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES", "",
		oID.OrderID)
	assert.NoError(t, err)
}

func TestGetFuturesFills(t *testing.T) {
	t.Parallel()
	_, err := bi.GetFuturesFills(context.Background(), 0, 0, 0, "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesFills(context.Background(), 0, 0, 0, "", "meow", time.Now(), time.Now().Add(-time.Minute))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesFills(context.Background(), 0, 1<<62, 5, "", testFiat2.String()+"-FUTURES", time.Time{},
		time.Time{})
	assert.NoError(t, err)
}

func TestGetPendingFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := bi.GetPendingFuturesOrders(context.Background(), 0, 0, 0, "", "", "", "", time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetPendingFuturesOrders(context.Background(), 0, 0, 0, "", "", "meow", "", time.Now(),
		time.Now().Add(-time.Minute))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetPendingFuturesOrders(context.Background(), 0, 1<<62, 5, "", "SBTCSUSDT",
		testFiat2.String()+"-FUTURES", "", time.Now().Add(-time.Hour*24*90), time.Now())
	assert.NoError(t, err)
}

func TestCommitConversion(t *testing.T) {
	// Not parallel due to collision with TestGetQuotedPrice
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
	assert.NoError(t, err)
	_, err = bi.CommitConversion(context.Background(), testCrypto.String(), testFiat.String(), resp.Data.TraceID,
		resp.Data.FromCoinSize, resp.Data.ToCoinSize, resp.Data.ConvertPrice)
	assert.NoError(t, err)
}

// The following 4 tests aren't parallel due to collisions with each other, and some other futures-related tests
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
	oID := getFuturesOrdIDHelper(t)
	cID := clientIDGenerator()
	_, err = bi.ModifyFuturesOrder(context.Background(), oID.OrderID, oID.ClientOrderID, testPair2.String(),
		testFiat2.String()+"-FUTURES", cID, 0, 0, testAmount2+1, testPrice2)
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
	oID := getFuturesOrdIDHelper(t)
	_, err = bi.CancelFuturesOrder(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES", "",
		"", oID.OrderID)
	assert.NoError(t, err)
}

func TestBatchCancelFuturesOrders(t *testing.T) {
	_, err := bi.BatchCancelFuturesOrders(context.Background(), nil, "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	oID := getFuturesOrdIDHelper(t)
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
	_ = placeFuturesOrderHelper(t)
	_, err = bi.FlashClosePosition(context.Background(), testPair2.String(), "", testFiat2.String()+"-FUTURES")
	assert.NoError(t, err)
}

type getNoArgsResp interface {
	*TimeResp | *P2PMerInfoResp | *ConvertCoinsResp | *BGBConvertCoinsResp | *VIPFeeRateResp
}

type getNoArgsAssertNotEmpty[G getNoArgsResp] func(context.Context) (G, error)

func testGetNoArgs[G getNoArgsResp](t *testing.T, f getNoArgsAssertNotEmpty[G]) {
	t.Helper()
	_, err := f(context.Background())
	assert.NoError(t, err)
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

func getPlanOrdIDHelper(t *testing.T, mustBeTriggered bool) OrderIDStruct {
	t.Helper()
	var ordIDs OrderIDStruct
	resp, err := bi.GetCurrentPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 100,
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

func placeFuturesOrderHelper(t *testing.T) *OrderResp {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	cID := clientIDGenerator()
	resp, err := bi.PlaceFuturesOrder(context.Background(), testPair2.String(), testFiat2.String()+"-FUTURES",
		"isolated", testFiat2.String(), "buy", "open", "market", "GTC", cID, "", "", testAmount2, testPrice2, true,
		true)
	if err != nil {
		if strings.Contains(err.Error(), errAmountExceedsBalancePartial) {
			t.Skip(skipInsufficientBalance)
		} else {
			t.Error(err)
		}
	}
	return resp
}

func getFuturesOrdIDHelper(t *testing.T) *OrderIDStruct {
	resp, err := bi.GetPendingFuturesOrders(context.Background(), 0, 1<<62, 5, "", "SBTCSUSDT",
		testFiat2.String()+"-FUTURES", "", time.Now().Add(-time.Hour*24*90), time.Now())
	if err != nil {
		t.Error(err)
	}
	if resp != nil {
		if len(resp.Data.EntrustedList) != 0 {
			return &OrderIDStruct{
				OrderID:       resp.Data.EntrustedList[0].OrderID,
				ClientOrderID: resp.Data.EntrustedList[0].ClientOrderID,
			}
		}
	}
	IDs := placeFuturesOrderHelper(t)
	return &OrderIDStruct{
		OrderID:       int64(IDs.Data.OrderID),
		ClientOrderID: IDs.Data.ClientOrderID,
	}
}

func clientIDGenerator() string {
	i := time.Now().UnixNano()>>29 + time.Now().UnixNano()<<35
	cID := testSubaccountName + strconv.FormatInt(i, 10)
	if len(cID) > 50 {
		cID = cID[:50]
	}
	return cID
}

func aBenchmarkHelper(a, pag int64) {
	var params Params
	params.Values = make(url.Values)
	params.Values.Set("limit", strconv.FormatInt(a, 10))
	params.Values.Set("idLessThan", strconv.FormatInt(pag, 10))
}

// 4005735	       289.4 ns/op	      24 B/op	       1 allocs/op
// 4498795	       250.1 ns/op	      24 B/op	       1 allocs/op
// 4661677	       258.9 ns/op	      24 B/op	       1 allocs/op
// 3078195	       376.8 ns/op	      24 B/op	       1 allocs/op
// 3179569	       333.8 ns/op	      24 B/op	       1 allocs/op

// 3315597	       318.2 ns/op	      24 B/op	       1 allocs/op
// 3386540	       342.5 ns/op	      24 B/op	       1 allocs/op
// 3463975	       313.6 ns/op	      24 B/op	       1 allocs/op

// 3449566	       294.4 ns/op	      24 B/op	       1 allocs/op
// 4315550	       279.4 ns/op	      24 B/op	       1 allocs/op

// 2733825	       386.7 ns/op	      23 B/op	       1 allocs/op
// 4069201	       296.4 ns/op	      23 B/op	       1 allocs/op
// 3914810	       315.4 ns/op	      23 B/op	       1 allocs/op
func BenchmarkGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		g := time.Now().UnixNano()>>29 + time.Now().UnixNano()<<35
		cID := testSubaccountName + strconv.FormatInt(g, 10)
		if len(cID) > 50 {
			cID = cID[:50]
		}
	}
}

func TestSilly(t *testing.T) {
	i := 51
	var f float64
	f = 1 << 511
	log.Print(f)
	f = float64(i << i)
	log.Print(f)
	i = 100
	f = float64(i << i)
	log.Print(f)
	i = 512
	f = float64(i << i)
	log.Print(f)
}
