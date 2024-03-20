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

	testSubAccountName = "GCTTESTA"
)

// User-defined variables to aid testing
var (
	testCrypto  = currency.BTC
	testCrypto2 = currency.DOGE // Used for endpoints which consume all available funds
	testFiat    = currency.USDT
	testPair    = currency.NewPair(testCrypto, testFiat)
	testIP      = "14.203.57.50"
	testAmount  = 0.00001
	testPrice   = 1e9
)

// Developer-defined constants to aid testing
const (
	skipTestSubAccNotFound       = "test sub-account not found, skipping"
	skipInsufficientAPIKeysFound = "insufficient API keys found, skipping"
	skipInsufficientBalance      = "insufficient balance to place order, skipping"

	errorAPIKeyLimitPartial         = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"40063","msg":"API exceeds the maximum limit added","requestTime":`
	errorInsufficientBalancePartial = `Bitget unsuccessful HTTP status code: 400 raw response: {"code":"43012","msg":"Insufficient balance","requestTime":`
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
	_, err = bi.GetSpotTransactionRecords(context.Background(), "", time.Now().Add(-time.Hour*24*30), time.Now(), 0, 0)
	assert.NoError(t, err)
}

func TestGetFuturesTransactionRecords(t *testing.T) {
	_, err := bi.GetFuturesTransactionRecords(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errProductTypeEmpty)
	_, err = bi.GetFuturesTransactionRecords(context.Background(), "woof", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetFuturesTransactionRecords(context.Background(), "COIN-FUTURES", "",
		time.Now().Add(-time.Hour*24*30), time.Now(), 0, 0)
	assert.NoError(t, err)
}

func TestGetMarginTransactionRecords(t *testing.T) {
	_, err := bi.GetMarginTransactionRecords(context.Background(), "", "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetMarginTransactionRecords(context.Background(), "", "", time.Now().Add(-time.Hour*24*30), time.Now(), 0, 0)
	assert.NoError(t, err)
}

func TestGetP2PTransactionRecords(t *testing.T) {
	_, err := bi.GetP2PTransactionRecords(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetP2PTransactionRecords(context.Background(), "", time.Now().Add(-time.Hour*24*30), time.Now(), 0, 0)
	assert.NoError(t, err)
}

func TestGetP2PMerchantList(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	// Can't currently be properly tested due to not knowing any p2p merchant IDs
	_, err := bi.GetP2PMerchantList(context.Background(), "", "1", 0, 0)
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
	_, err = bi.GetMerchantP2POrders(context.Background(), time.Now().Add(-time.Hour*24*30), time.Now(), 0, 0, 0,
		0, "", "", "", "")
	assert.NoError(t, err)
}

func TestGetMerchantAdvertisementList(t *testing.T) {
	_, err := bi.GetMerchantAdvertisementList(context.Background(), time.Time{}, time.Time{}, 0, 0, 0, 0, "", "", "",
		"", "", "")
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetMerchantAdvertisementList(context.Background(), time.Now().Add(-time.Hour*24*30), time.Now(), 0,
		0, 0, 0, "", "", "", "", "", "")
	assert.NoError(t, err)
}

func TestCreateVirtualSubaccounts(t *testing.T) {
	_, err := bi.CreateVirtualSubaccounts(context.Background(), []string{})
	assert.ErrorIs(t, err, errSubAccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	resp, err := bi.CreateVirtualSubaccounts(context.Background(), []string{testSubAccountName})
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestModifyVirtualSubaccount(t *testing.T) {
	perms := []string{}
	_, err := bi.ModifyVirtualSubaccount(context.Background(), "", "", perms)
	assert.ErrorIs(t, err, errSubAccountEmpty)
	_, err = bi.ModifyVirtualSubaccount(context.Background(), "meow", "", perms)
	assert.ErrorIs(t, err, errNewStatusEmpty)
	_, err = bi.ModifyVirtualSubaccount(context.Background(), "meow", "woof", perms)
	assert.ErrorIs(t, err, errNewPermsEmpty)
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
	tarID := subAccTestHelper(t)
	perms = append(perms, "read")
	resp, err := bi.ModifyVirtualSubaccount(context.Background(), tarID, "normal", perms)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestCreateSubaccountAndAPIKey(t *testing.T) {
	ipL := []string{}
	_, err := bi.CreateSubaccountAndAPIKey(context.Background(), "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errSubAccountEmpty)
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
	ipL = append(ipL, testIP)
	pL := []string{"read"}
	// Fails with error "subAccountList not empty" and I'm not sure why. The account I'm testing with is far off
	// hitting the limit of 20 sub-accounts.
	_, err = bi.CreateSubaccountAndAPIKey(context.Background(), "MEOWMEOW", "woofwoof", "neighneighneighneighneigh",
		ipL, pL)
	assert.NoError(t, err)
}

func TestGetVirtualSubaccounts(t *testing.T) {
	_, err := bi.GetVirtualSubaccounts(context.Background(), 25, 0, "")
	assert.NoError(t, err)
}

func TestCreateAPIKey(t *testing.T) {
	ipL := []string{}
	_, err := bi.CreateAPIKey(context.Background(), "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errSubAccountEmpty)
	_, err = bi.CreateAPIKey(context.Background(), "woof", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errPassphraseEmpty)
	_, err = bi.CreateAPIKey(context.Background(), "woof", "meow", "", ipL, ipL)
	assert.ErrorIs(t, err, errLabelEmpty)
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
	tarID := subAccTestHelper(t)
	ipL = append(ipL, testIP)
	pL := []string{"read"}
	_, err = bi.CreateAPIKey(context.Background(), tarID, clientID, "neigh whinny", ipL, pL)
	if err != nil && !strings.Contains(err.Error(), errorAPIKeyLimitPartial) {
		t.Error(err)
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
	assert.ErrorIs(t, err, errSubAccountEmpty)
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
	tarID := subAccTestHelper(t)
	resp, err := bi.GetAPIKeys(context.Background(), tarID)
	assert.NoError(t, err)
	if len(resp.Data) == 0 {
		t.Skip(skipInsufficientAPIKeysFound)
	}
	resp2, err := bi.ModifyAPIKey(context.Background(), tarID, clientID, "oink", resp.Data[0].SubAccountApiKey,
		ipL, ipL)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp2)
}

func TestGetAPIKeys(t *testing.T) {
	_, err := bi.GetAPIKeys(context.Background(), "")
	assert.ErrorIs(t, err, errSubAccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := subAccTestHelper(t)
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
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
	resp, err := bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), testAmount, 0)
	assert.NoError(t, err)
	_, err = bi.CommitConversion(context.Background(), testCrypto.String(), testFiat.String(), resp.Data.TraceID,
		resp.Data.FromCoinSize, resp.Data.ToCoinSize, resp.Data.ConvertPrice)
	if err != nil && !strings.Contains(err.Error(), errorInsufficientBalancePartial) {
		t.Error(err)
	}
}

func TestGetConvertHistory(t *testing.T) {
	_, err := bi.GetConvertHistory(context.Background(), time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, common.ErrDateUnset)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetConvertHistory(context.Background(), time.Now().Add(-time.Hour*90*24), time.Now(), 1, 0)
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
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
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
	_, err = bi.GetBGBConvertHistory(context.Background(), 0, 1, 0, time.Time{}, time.Time{})
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

func TestGetVIPFeeRate(t *testing.T) {
	testGetNoArgs(t, bi.GetVIPFeeRate)
}

func TestGetTickerInformation(t *testing.T) {
	resp, err := bi.GetTickerInformation(context.Background(), testPair.String())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetMergeDepth(t *testing.T) {
	_, err := bi.GetMergeDepth(context.Background(), "", "", "")
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetMergeDepth(context.Background(), testPair.String(), "scale3", "5")
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetOrderbookDepth(t *testing.T) {
	resp, err := bi.GetOrderbookDepth(context.Background(), testPair.String(), "", 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetCandlestickData(t *testing.T) {
	_, err := bi.GetCandlestickData(context.Background(), "", "", time.Time{}, time.Time{}, 0, false)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetCandlestickData(context.Background(), "meow", "", time.Time{}, time.Time{}, 0, false)
	assert.ErrorIs(t, err, errGranEmpty)
	_, err = bi.GetCandlestickData(context.Background(), "meow", "woof", time.Time{}, time.Time{}, 5,
		true)
	assert.ErrorIs(t, err, errEndTimeEmpty)
	_, err = bi.GetCandlestickData(context.Background(), "meow", "woof", time.Now(), time.Now().Add(-time.Second), 0,
		false)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	resp, err := bi.GetCandlestickData(context.Background(), testPair.String(), "1min", time.Time{}, time.Time{}, 5,
		false)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
	resp, err = bi.GetCandlestickData(context.Background(), testPair.String(), "1min", time.Time{}, time.Now(), 5,
		true)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetRecentFills(t *testing.T) {
	_, err := bi.GetRecentFills(context.Background(), "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	resp, err := bi.GetRecentFills(context.Background(), testPair.String(), 5)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
}

func TestGetMarketTrades(t *testing.T) {
	_, err := bi.GetMarketTrades(context.Background(), "", time.Time{}, time.Time{}, 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetMarketTrades(context.Background(), "meow", time.Now(), time.Now().Add(-time.Second), 0, 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd)
	resp, err := bi.GetMarketTrades(context.Background(), testPair.String(), time.Time{}, time.Time{}, 5,
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
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
	_, err = bi.PlaceOrder(context.Background(), testPair.String(), "sell", "limit", "IOC", "", testPrice, testAmount)
	if err != nil && !strings.Contains(err.Error(), errorInsufficientBalancePartial) {
		t.Error(err)
	}
}

func TestCancelOrderByID(t *testing.T) {
	_, err := bi.CancelOrderByID(context.Background(), "", "", 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.CancelOrderByID(context.Background(), testPair.String(), "", 0)
	assert.ErrorIs(t, err, errOrderClientEmpty)
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
	resp, err := bi.PlaceOrder(context.Background(), testPair.String(), "sell", "limit", "GTC", "", testPrice,
		testAmount)
	if strings.Contains(err.Error(), errorInsufficientBalancePartial) {
		t.Skip(skipInsufficientBalance)
	}
	if err != nil {
		t.Error(err)
	}
	_, err = bi.CancelOrderByID(context.Background(), testPair.String(), "", resp.Data.OrderID)
	if err != nil {
		t.Error(err)
	}
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

func subAccTestHelper(t *testing.T) string {
	t.Helper()
	resp, err := bi.GetVirtualSubaccounts(context.Background(), 25, 0, "")
	assert.NoError(t, err)
	tarID := ""
	compString := strings.ToLower(string(testSubAccountName[:3])) + "****@virtual-bitget.com"
	for i := range resp.Data.SubAccountList {
		if resp.Data.SubAccountList[i].SubAccountName == compString {
			tarID = resp.Data.SubAccountList[i].SubAccountUID
			break
		}
	}
	if tarID == "" {
		t.Skip(skipTestSubAccNotFound)
	}
	return tarID
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
