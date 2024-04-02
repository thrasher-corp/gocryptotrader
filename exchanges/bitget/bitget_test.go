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

	testSubaccountName = "GCTTESTA"
	testIP             = "14.203.57.50"
	testAmount         = 0.00001
	testPrice          = 1e9
	// Donation address by default
	testAddress = "bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc"
)

// User-defined variables to aid testing
var (
	testCrypto  = currency.BTC
	testCrypto2 = currency.DOGE // Used for endpoints which consume all available funds
	testFiat    = currency.USDT
	testPair    = currency.NewPair(testCrypto, testFiat)
)

// Developer-defined constants to aid testing
const (
	skipTestSubAccNotFound       = "appropriate sub-account (equals %v, not equals %v) not found, skipping"
	skipInsufficientAPIKeysFound = "insufficient API keys found, skipping"
	skipInsufficientBalance      = "insufficient balance to place order, skipping"

	errorAPIKeyLimitPartial = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"40063","msg":"API exceeds the maximum limit added","requestTime":`
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
	var e exchange.IBotExchange
	if e = new(Bitget); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func TestQueryAnnouncements(t *testing.T) {
	_, err := bi.QueryAnnouncements(context.Background(), "", time.Now().Add(time.Hour), time.Now())
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	resp, err := bi.QueryAnnouncements(context.Background(), "latest_news", time.Time{}, time.Time{})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetTime(t *testing.T) {
	testGetNoArgs(t, bi.GetTime)
}

func TestGetTradeRate(t *testing.T) {
	_, err := bi.GetTradeRate(context.Background(), "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetTradeRate(context.Background(), "BTCUSDT", "")
	assert.ErrorIs(t, err, errBusinessTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetTradeRate(context.Background(), "BTCUSDT", "spot")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotTransactionRecords(t *testing.T) {
	_, err := bi.GetSpotTransactionRecords(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotTransactionRecords(context.Background(), "", time.Now().Add(-time.Hour*24*30), time.Now(), 5,
		1<<62)
	assert.NoError(t, err)
}

func TestGetFuturesTransactionRecords(t *testing.T) {
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
	_, err := bi.GetMarginTransactionRecords(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetMarginTransactionRecords(context.Background(), "", "", time.Now().Add(-time.Hour*24*30),
		time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetP2PTransactionRecords(t *testing.T) {
	_, err := bi.GetP2PTransactionRecords(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetP2PTransactionRecords(context.Background(), "", time.Now().Add(-time.Hour*24*30), time.Now(), 5,
		1<<62)
	assert.NoError(t, err)
}

func TestGetP2PMerchantList(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// Can't currently be properly tested due to not knowing any p2p merchant IDs
	_, err := bi.GetP2PMerchantList(context.Background(), "", "1", 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetMerchantInfo(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetMerchantInfo)
}

func TestGetMerchantP2POrders(t *testing.T) {
	_, err := bi.GetMerchantP2POrders(context.Background(), time.Time{}, time.Time{}, 0, 0, 0, 0, "", "", "", "")
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// Can't currently be properly tested due to not knowing any p2p order IDs
	_, err = bi.GetMerchantP2POrders(context.Background(), time.Now().Add(-time.Hour*24*30), time.Now(), 5, 1<<62, 0,
		0, "", "", "", "")
	assert.NoError(t, err)
}

func TestGetMerchantAdvertisementList(t *testing.T) {
	_, err := bi.GetMerchantAdvertisementList(context.Background(), time.Time{}, time.Time{}, 0, 0, 0, 0, "", "", "",
		"", "", "")
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetMerchantAdvertisementList(context.Background(), time.Now().Add(-time.Hour*24*30), time.Now(), 5,
		1<<62, 0, 0, "", "", "", "", "", "")
	assert.NoError(t, err)
}

func TestCreateVirtualSubaccounts(t *testing.T) {
	_, err := bi.CreateVirtualSubaccounts(context.Background(), []string{})
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.CreateVirtualSubaccounts(context.Background(), []string{testSubaccountName})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyVirtualSubaccount(t *testing.T) {
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
	assert.NotEmpty(t, resp)
}

func TestCreateSubaccountAndAPIKey(t *testing.T) {
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
	_, err := bi.GetVirtualSubaccounts(context.Background(), 25, 1<<62, "")
	assert.NoError(t, err)
}

func TestCreateAPIKey(t *testing.T) {
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
		if !strings.Contains(err.Error(), errorAPIKeyLimitPartial) {
			t.Log(err)
		} else {
			t.Error(err)
		}
	}
}

func TestModifyAPIKey(t *testing.T) {
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
	assert.NotEmpty(t, resp2)
}

func TestGetAPIKeys(t *testing.T) {
	_, err := bi.GetAPIKeys(context.Background(), "")
	assert.ErrorIs(t, err, errSubaccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := subAccTestHelper(t, strings.ToLower(string(testSubaccountName[:3]))+"****@virtual-bitget.com", "")
	resp, err := bi.GetAPIKeys(context.Background(), tarID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetConvertCoints(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetConvertCoins)
}

func TestGetQuotedPrice(t *testing.T) {
	_, err := bi.GetQuotedPrice(context.Background(), "", "", 0, 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.GetQuotedPrice(context.Background(), "meow", "woof", 0, 0)
	assert.ErrorIs(t, err, errFromToMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), 0, 1)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
	resp, err = bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), 0.1, 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestCommitConversion(t *testing.T) {
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

func TestGetConvertHistory(t *testing.T) {
	_, err := bi.GetConvertHistory(context.Background(), time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetConvertHistory(context.Background(), time.Now().Add(-time.Hour*90*24), time.Now(), 5, 1<<62)
	assert.NoError(t, err)
}

func TestGetBGBConvertCoins(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	testGetNoArgs(t, bi.GetBGBConvertCoins)
}

func TestConvertBGB(t *testing.T) {
	var currencies []string
	_, err := bi.ConvertBGB(context.Background(), currencies)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	currencies = append(currencies, testCrypto2.String())
	// No matter what currency I use, this returns the error "currency does not support convert"; possibly a bad error
	// message, with the true issue being lack of funds?
	_, err = bi.ConvertBGB(context.Background(), currencies)
	assert.NoError(t, err)
}

func TestGetBGBConvertHistory(t *testing.T) {
	_, err := bi.GetBGBConvertHistory(context.Background(), 0, 0, 0, time.Now(), time.Now().Add(-time.Second))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetBGBConvertHistory(context.Background(), 0, 5, 1<<62, time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetCoinInfo(t *testing.T) {
	resp, err := bi.GetCoinInfo(context.Background(), "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSymbolInfo(t *testing.T) {
	resp, err := bi.GetSymbolInfo(context.Background(), "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotVIPFeeRate(t *testing.T) {
	testGetNoArgs(t, bi.GetSpotVIPFeeRate)
}

func TestGetSpotTickerInformation(t *testing.T) {
	resp, err := bi.GetSpotTickerInformation(context.Background(), testPair.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotMergeDepth(t *testing.T) {
	_, err := bi.GetSpotMergeDepth(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetSpotMergeDepth(context.Background(), testPair.String(), "scale3", "5")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetOrderbookDepth(t *testing.T) {
	resp, err := bi.GetOrderbookDepth(context.Background(), testPair.String(), "", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotCandlestickData(t *testing.T) {
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
	assert.NotEmpty(t, resp)
	resp, err = bi.GetSpotCandlestickData(context.Background(), testPair.String(), "1min", time.Time{}, time.Now(), 5,
		true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetRecentSpotFills(t *testing.T) {
	_, err := bi.GetRecentSpotFills(context.Background(), "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetRecentSpotFills(context.Background(), testPair.String(), 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotMarketTrades(t *testing.T) {
	_, err := bi.GetSpotMarketTrades(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetSpotMarketTrades(context.Background(), "meow", time.Now(), time.Now().Add(-time.Second), 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	resp, err := bi.GetSpotMarketTrades(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5,
		1<<63-1)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestPlaceOrder(t *testing.T) {
	_, err := bi.PlaceOrder(context.Background(), "", "", "", "", "", 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.PlaceOrder(context.Background(), testPair.String(), "", "", "", "", 0, 0)
	assert.ErrorIs(t, err, errSideEmpty)
	_, err = bi.PlaceOrder(context.Background(), testPair.String(), "sell", "", "", "", 0, 0)
	assert.ErrorIs(t, err, errOrderTypeEmpty)
	_, err = bi.PlaceOrder(context.Background(), testPair.String(), "sell", "limit", "", "", 0, 0)
	assert.ErrorIs(t, err, errStrategyEmpty)
	_, err = bi.PlaceOrder(context.Background(), testPair.String(), "sell", "limit", "IOC", "", 0, 0)
	assert.ErrorIs(t, err, errLimitPriceEmpty)
	_, err = bi.PlaceOrder(context.Background(), testPair.String(), "sell", "limit", "IOC", "", testPrice, 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceOrder(context.Background(), testPair.String(), "sell", "limit", "IOC", "", testPrice, testAmount)
	assert.NoError(t, err)
}

func TestCancelOrderByID(t *testing.T) {
	_, err := bi.CancelOrderByID(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelOrderByID(context.Background(), testPair.String(), "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.CancelOrderByID(context.Background(), "meow", "a", -1)
	if err == nil {
		t.Error(errNonsenseRequest)
	}
	cID := testSubaccountName + strconv.FormatInt(time.Now().Unix(), 10)
	if len(cID) > 50 {
		cID = cID[:50]
	}
	resp, err := bi.PlaceOrder(context.Background(), testPair.String(), "sell", "limit", "GTC", cID, testPrice,
		testAmount)
	require.NoError(t, err)
	_, err = bi.CancelOrderByID(context.Background(), testPair.String(), resp.Data.ClientOrderID,
		int64(resp.Data.OrderID))
	assert.NoError(t, err)
}

func TestBatchPlaceOrder(t *testing.T) {
	var req []PlaceOrderStruct
	_, err := bi.BatchPlaceOrder(context.Background(), "", req)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.BatchPlaceOrder(context.Background(), "meow", req)
	assert.ErrorIs(t, err, errOrdersEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	req = append(req, PlaceOrderStruct{
		Side:      "sell",
		OrderType: "limit",
		Strategy:  "IOC",
		Price:     testPrice,
		Size:      testAmount,
	})
	resp, err := bi.BatchPlaceOrder(context.Background(), testPair.String(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestBatchCancelOrders(t *testing.T) {
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
	resp, err := bi.PlaceOrder(context.Background(), testPair.String(), "sell", "limit", "IOC", "", testPrice, testAmount)
	require.NoError(t, err)
	req = append(req, OrderIDStruct{
		OrderID:       int64(resp.Data.OrderID),
		ClientOrderID: resp.Data.ClientOrderID,
	})
	resp2, err := bi.BatchCancelOrders(context.Background(), testPair.String(), req)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2)
}

func TestCancelOrderBySymbol(t *testing.T) {
	_, err := bi.CancelOrderBySymbol(context.Background(), "")
	assert.ErrorIs(t, err, errPairEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.CancelOrderBySymbol(context.Background(), testPair.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetOrderDetails(t *testing.T) {
	_, err := bi.GetOrderDetails(context.Background(), 0, "")
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	ordIDs := getPlanOrdIDHelper(t)
	_, err = bi.GetOrderDetails(context.Background(), ordIDs.OrderID, ordIDs.ClientOrderID)
	assert.NoError(t, err)
}

func TestGetUnfilledOrders(t *testing.T) {
	_, err := bi.GetUnfilledOrders(context.Background(), "", time.Now(), time.Now().Add(-time.Second), 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetUnfilledOrders(context.Background(), "", time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetHistoricalOrders(t *testing.T) {
	_, err := bi.GetHistoricalOrders(context.Background(), "", time.Now(), time.Now().Add(-time.Second), 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetHistoricalOrders(context.Background(), "", time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFills(t *testing.T) {
	_, err := bi.GetFills(context.Background(), "", time.Time{}, time.Time{}, 0, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFills(context.Background(), "meow", time.Now(), time.Now().Add(-time.Second), 0, 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFills(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62, 0)
	assert.NoError(t, err)
}

func TestPlacePlanOrder(t *testing.T) {
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
	cID := testSubaccountName + strconv.FormatInt(time.Now().Unix(), 10)
	if len(cID) > 50 {
		cID = cID[:50]
	}
	resp, err := bi.PlacePlanOrder(context.Background(), testPair.String(), "sell", "limit", "", "fill_price", cID,
		"ioc", testPrice, testPrice, testAmount)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyPlanOrder(t *testing.T) {
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
	ordID, err := bi.PlacePlanOrder(context.Background(), testPair.String(), "sell", "limit", "", "fill_price", "",
		"ioc", testPrice, testPrice, testAmount)
	assert.NoError(t, err)
	require.NotEmpty(t, ordID)
	resp, err := bi.ModifyPlanOrder(context.Background(), ordID.Data.OrderID, ordID.Data.ClientOrderID, "limit",
		testPrice, testPrice, testAmount)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestCancelPlanOrder(t *testing.T) {
	_, err := bi.CancelPlanOrder(context.Background(), 0, "")
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	ordID, err := bi.PlacePlanOrder(context.Background(), testPair.String(), "sell", "limit", "", "fill_price", "",
		"ioc", testPrice, testPrice, testAmount)
	assert.NoError(t, err)
	require.NotEmpty(t, ordID)
	resp, err := bi.CancelPlanOrder(context.Background(), ordID.Data.OrderID, ordID.Data.ClientOrderID)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCurrentPlanOrders(t *testing.T) {
	_, err := bi.GetCurrentPlanOrders(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCurrentPlanOrders(context.Background(), "meow", time.Now(), time.Now().Add(-time.Second), 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetCurrentPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5, 1<<62)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetPlanSubOrder(t *testing.T) {
	_, err := bi.GetPlanSubOrder(context.Background(), "")
	assert.ErrorIs(t, err, errOrderIDEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	ordIDs := getPlanOrdIDHelper(t)
	resp, err := bi.GetPlanSubOrder(context.Background(), strconv.FormatInt(ordIDs.OrderID, 10))
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetPlanOrderHistory(t *testing.T) {
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
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.BatchCancelPlanOrders(context.Background(), []string{testPair.String()})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAccountInfo(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetAccountInfo(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAccountAssets(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetAccountAssets(context.Background(), "", "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSpotSubaccountAssets(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetSpotSubaccountAssets(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyDepositAccount(t *testing.T) {
	_, err := bi.ModifyDepositAccount(context.Background(), "", "")
	assert.ErrorIs(t, err, errAccountTypeEmpty)
	_, err = bi.ModifyDepositAccount(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ModifyDepositAccount(context.Background(), "spot", testFiat.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAccountBills(t *testing.T) {
	_, err := bi.GetSpotAccountBills(context.Background(), "", "", "", time.Now(), time.Now().Add(-time.Minute), 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSpotAccountBills(context.Background(), testCrypto.String(), "", "", time.Time{}, time.Time{}, 3, 1<<62)
	assert.NoError(t, err)
}

func TestTransferAsset(t *testing.T) {
	_, err := bi.TransferAsset(context.Background(), "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.TransferAsset(context.Background(), "meow", "", "", "", "", 0)
	assert.ErrorIs(t, err, errToTypeEmpty)
	_, err = bi.TransferAsset(context.Background(), "meow", "woof", "", "", "", 0)
	assert.ErrorIs(t, err, errCurrencyAndPairEmpty)
	_, err = bi.TransferAsset(context.Background(), "meow", "woof", "neigh", "", "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	cID := testSubaccountName + strconv.FormatInt(time.Now().Unix(), 10)
	if len(cID) > 50 {
		cID = cID[:50]
	}
	_, err = bi.TransferAsset(context.Background(), "spot", "p2p", testCrypto.String(), testPair.String(), cID,
		testAmount)
	assert.NoError(t, err)
}

func TestGetTransferableCoinList(t *testing.T) {
	_, err := bi.GetTransferableCoinList(context.Background(), "", "")
	assert.ErrorIs(t, err, errFromTypeEmpty)
	_, err = bi.GetTransferableCoinList(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errToTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetTransferableCoinList(context.Background(), "spot", "p2p")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestSubaccountTransfer(t *testing.T) {
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
	cID := testSubaccountName + strconv.FormatInt(time.Now().Unix(), 10)
	if len(cID) > 50 {
		cID = cID[:50]
	}
	_, err = bi.SubaccountTransfer(context.Background(), "spot", "p2p", testCrypto.String(), testPair.String(), cID,
		fromID, toID, testAmount)
	assert.NoError(t, err)
}

func TestWithdrawFunds(t *testing.T) {
	_, err := bi.WithdrawFunds(context.Background(), "", "", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errCurrencyEmpty)
	_, err = bi.WithdrawFunds(context.Background(), "meow", "", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errTransferTypeEmpty)
	_, err = bi.WithdrawFunds(context.Background(), "meow", "woof", "", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errAddressEmpty)
	_, err = bi.WithdrawFunds(context.Background(), "meow", "woof", "neigh", "", "", "", "", "", "", 0)
	assert.ErrorIs(t, err, errAmountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	_, err = bi.WithdrawFunds(context.Background(), testCrypto.String(), "on_chain", testAddress, "", "", "", "", "",
		"", testAmount)
	assert.NoError(t, err)
}

func TestGetSubaccountTransferRecord(t *testing.T) {
	_, err := bi.GetSubaccountTransferRecord(context.Background(), "", "", "", time.Now(), time.Now().Add(-time.Minute),
		0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetSubaccountTransferRecord(context.Background(), "", "", "meow", time.Time{}, time.Time{}, 3, 1<<62)
	assert.NoError(t, err)
}

func TestGetTransferRecord(t *testing.T) {
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
	_, err := bi.GetDepositAddressForCurrency(context.Background(), "", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetDepositAddressForCurrency(context.Background(), testCrypto.String(), "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSubaccountDepositAddress(t *testing.T) {
	_, err := bi.GetSubaccountDepositAddress(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errSubaccountEmpty)
	_, err = bi.GetSubaccountDepositAddress(context.Background(), "meow", "", "")
	assert.ErrorIs(t, err, errCurrencyEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := subAccTestHelper(t, "", "")
	resp, err := bi.GetSubaccountDepositAddress(context.Background(), tarID, testCrypto.String(), "")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetSubaccountDepositRecords(t *testing.T) {
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
	_, err := bi.GetDepositRecords(context.Background(), "", 0, 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetDepositRecords(context.Background(), testCrypto.String(), 0, 1<<62, 2,
		time.Now().Add(-time.Hour*24*90), time.Now())
	assert.NoError(t, err)
}

func TestGetFuturesMergeDepth(t *testing.T) {
	_, err := bi.GetFuturesMergeDepth(context.Background(), "", "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFuturesMergeDepth(context.Background(), "meow", "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetFuturesMergeDepth(context.Background(), testPair.String(), "USDT-FUTURES", "scale3", "5")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesVIPFeeRate(t *testing.T) {
	testGetNoArgs(t, bi.GetFuturesVIPFeeRate)
}

func TestGetFuturesTicker(t *testing.T) {
	testGetTwoArgs(t, bi.GetFuturesTicker)
}

func TestGetAllFuturesTickers(t *testing.T) {
	_, err := bi.GetAllFuturesTickers(context.Background(), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetAllFuturesTickers(context.Background(), "COIN-FUTURES")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetRecentFuturesFills(t *testing.T) {
	_, err := bi.GetRecentFuturesFills(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetRecentFuturesFills(context.Background(), "meow", "", 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetRecentFuturesFills(context.Background(), testPair.String(), "USDT-FUTURES", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesCandlestickData(t *testing.T) {
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
	assert.NotEmpty(t, resp)
	resp, err = bi.GetFuturesCandlestickData(context.Background(), testPair.String(), "COIN-FUTURES", "1m",
		time.Time{}, time.Time{}, 5, CallModeHistory)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
	resp, err = bi.GetFuturesCandlestickData(context.Background(), testPair.String(), "USDC-FUTURES", "1m",
		time.Time{}, time.Now(), 5, CallModeIndex)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
	resp, err = bi.GetFuturesCandlestickData(context.Background(), testPair.String(), "USDT-FUTURES", "1m",
		time.Time{}, time.Now(), 5, CallModeMark)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetOpenPositions(t *testing.T) {
	testGetTwoArgs(t, bi.GetOpenPositions)
}

func TestGetNextFundingTime(t *testing.T) {
	testGetTwoArgs(t, bi.GetNextFundingTime)
}

func TestGetFuturesPrices(t *testing.T) {
	testGetTwoArgs(t, bi.GetFuturesPrices)
}

func TestGetFundingHistorical(t *testing.T) {
	_, err := bi.GetFundingHistorical(context.Background(), "", "", 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetFundingHistorical(context.Background(), "meow", "", 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	resp, err := bi.GetFundingHistorical(context.Background(), testPair.String(), "USDT-FUTURES", 5, 1)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFundingCurrent(t *testing.T) {
	testGetTwoArgs(t, bi.GetFundingCurrent)
}

func TestGetContractConfig(t *testing.T) {
	testGetTwoArgs(t, bi.GetContractConfig)
}

func TestGetOneFuturesAccount(t *testing.T) {
	_, err := bi.GetOneFuturesAccount(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetOneFuturesAccount(context.Background(), "meow", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetOneFuturesAccount(context.Background(), "meow", "woof", "")
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetOneFuturesAccount(context.Background(), testPair.String(), "USDT-FUTURES", "USDT")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetAllFuturesAccounts(t *testing.T) {
	_, err := bi.GetAllFuturesAccounts(context.Background(), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetAllFuturesAccounts(context.Background(), "COIN-FUTURES")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesSubaccountAssets(t *testing.T) {
	_, err := bi.GetFuturesSubaccountAssets(context.Background(), "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.GetFuturesSubaccountAssets(context.Background(), "USDT-FUTURES")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetEstimatedOpenCount(t *testing.T) {
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
	resp, err := bi.GetEstimatedOpenCount(context.Background(), testPair.String(), "USDT-FUTURES", "USDT", testPrice,
		testAmount, 20)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestChangeLeverage(t *testing.T) {
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
	assert.NotEmpty(t, resp)
}

func TestAdjustMargin(t *testing.T) {
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
	err = bi.AdjustMargin(context.Background(), testPair.String(), "USDT-FUTURES", "USDT", "long", -testAmount)
	assert.NoError(t, err)
}

func TestChangeMarginMode(t *testing.T) {
	_, err := bi.ChangeMarginMode(context.Background(), "", "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.ChangeMarginMode(context.Background(), "meow", "", "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ChangeMarginMode(context.Background(), "meow", "woof", "", "")
	assert.ErrorIs(t, err, errMarginCoinEmpty)
	_, err = bi.ChangeMarginMode(context.Background(), "meow", "woof", "neigh", "")
	assert.ErrorIs(t, err, errMarginModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ChangeMarginMode(context.Background(), testPair.String(), "USDT-FUTURES", "USDT", "crossed")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestChangePositionMode(t *testing.T) {
	_, err := bi.ChangePositionMode(context.Background(), "", "")
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.ChangePositionMode(context.Background(), "meow", "")
	assert.ErrorIs(t, err, errPositionModeEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
	resp, err := bi.ChangePositionMode(context.Background(), "USDT-FUTURES", "hedge_mode")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetFuturesAccountBills(t *testing.T) {
	_, err := bi.GetFuturesAccountBills(context.Background(), "", "", "", "", 0, 0, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesAccountBills(context.Background(), "meow", "", "", "", 0, 0, time.Now(),
		time.Now().Add(-time.Minute))
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesAccountBills(context.Background(), "USDT-FUTURES", "", "", "", 0, 0, time.Time{},
		time.Time{})
	assert.NoError(t, err)
}

type getNoArgsResp interface {
	*TimeResp | *P2PMerInfoResp | *ConvertCoinsResp | *BGBConvertCoinsResp | *VIPFeeRateResp
}

type getNoArgsAssertNotEmpty[G getNoArgsResp] func(context.Context) (G, error)

func testGetNoArgs[G getNoArgsResp](t *testing.T, f getNoArgsAssertNotEmpty[G]) {
	t.Helper()
	resp, err := f(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
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
	resp, err := f(context.Background(), testPair.String(), "USDT-FUTURES")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func subAccTestHelper(t *testing.T, compString, ignoreString string) string {
	t.Helper()
	resp, err := bi.GetVirtualSubaccounts(context.Background(), 25, 0, "")
	assert.NoError(t, err)
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

func getPlanOrdIDHelper(t *testing.T) OrderIDStruct {
	t.Helper()
	var ordIDs OrderIDStruct
	resp, err := bi.GetCurrentPlanOrders(context.Background(), testPair.String(), time.Time{}, time.Time{}, 1, 1<<62)
	if err != nil || len(resp.Data.OrderList) == 0 ||
		// If the first order's client ID is malformed for our purposes, don't bother digging deeper, and just place
		// a new, guaranteed safe, order
		resp.Data.OrderList[0].ClientOrderID != url.QueryEscape(resp.Data.OrderList[0].ClientOrderID) {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, bi, canManipulateRealOrders)
		resp2, err := bi.PlacePlanOrder(context.Background(), testPair.String(), "sell", "limit", "", "fill_price",
			"", "ioc", testPrice, testPrice, testAmount)
		require.NoError(t, err)
		require.NotEmpty(t, resp2)
		ordIDs.ClientOrderID = resp2.Data.ClientOrderID
		ordIDs.OrderID = resp2.Data.OrderID
	} else {
		ordIDs.ClientOrderID = resp.Data.OrderList[0].ClientOrderID
		ordIDs.OrderID = resp.Data.OrderList[0].OrderID
	}
	return ordIDs
}

func aBenchmarkHelper(a, pag int64) {
	var params Params
	params.Values = make(url.Values)
	params.Values.Set("limit", strconv.FormatInt(a, 10))
	params.Values.Set("idLessThan", strconv.FormatInt(pag, 10))
}

// 1790454	       664.8 ns/op	      60 B/op	       3 allocs/op
// 1966866	       637.5 ns/op	      60 B/op	       3 allocs/op
// 2024174	       520.1 ns/op	      60 B/op	       3 allocs/op
// 1555105	       662.2 ns/op	      60 B/op	       3 allocs/op
// 1770181	       666.8 ns/op	      60 B/op	       3 allocs/op
// 1774210	       722.2 ns/op	      60 B/op	       3 allocs/op
func BenchmarkGen(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = uint16(i) % 500
		g := int64(i) % 500
		aBenchmarkHelper(g, 5e7)
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
