package bitget

import (
	"context"
	"log"
	"os"
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
	canManipulateRealOrders = true
	testingInSandbox        = false

	testSubAccountName = "GCTTESTA"
)

// User-defined variables to aid testing
var (
	testCrypto = currency.BTC
	testFiat   = currency.USDT
	testPair   = currency.NewPair(testCrypto, testFiat)
)

// Developer-defined constants to aid testing
const (
	skipTestSubAccNotFound       = "test sub-account not found, skipping"
	skipInsufficientAPIKeysFound = "insufficient API keys found, skipping"

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
	resp, err := bi.GetTime(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, resp)
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
	_, err := bi.GetMerchantInfo(context.Background())
	assert.NoError(t, err)
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
	_, err = bi.CreateVirtualSubaccounts(context.Background(), []string{testSubAccountName})
	assert.NoError(t, err)
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
	_, err = bi.ModifyVirtualSubaccount(context.Background(), tarID, "normal", perms)
	assert.NoError(t, err)
}

func TestCreateSubaccountAndAPIKey(t *testing.T) {
	ipL := []string{}
	_, err := bi.CreateSubaccountAndAPIKey(context.Background(), "", "", "", ipL, ipL)
	assert.ErrorIs(t, err, errSubAccountEmpty)
	sharedtestvalues.SkipTestIfCannotManipulateOrders(t, bi, canManipulateRealOrders)
	ipL = append(ipL, "14.203.57.50")
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
	ipL = append(ipL, "14.203.57.50")
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
	_, err = bi.ModifyAPIKey(context.Background(), tarID, clientID, "oink", resp.Data[0].SubAccountApiKey,
		ipL, ipL)
	assert.NoError(t, err)
}

func TestGetAPIKeys(t *testing.T) {
	_, err := bi.GetAPIKeys(context.Background(), "")
	assert.ErrorIs(t, err, errSubAccountEmpty)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	tarID := subAccTestHelper(t)
	_, err = bi.GetAPIKeys(context.Background(), tarID)
	assert.NoError(t, err)
}

func TestGetConvertCoints(t *testing.T) {
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err := bi.GetConvertCoins(context.Background())
	assert.NoError(t, err)
}

func TestGetQuotedPrice(t *testing.T) {
	_, err := bi.GetQuotedPrice(context.Background(), "", "", 0, 0)
	assert.ErrorIs(t, err, errPairEmpty)
	_, err = bi.GetQuotedPrice(context.Background(), "meow", "woof", 0, 0)
	assert.ErrorIs(t, err, errFromToMutex)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, bi)
	_, err = bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), 0, 1)
	assert.NoError(t, err)
	_, err = bi.GetQuotedPrice(context.Background(), testCrypto.String(), testFiat.String(), 0.1, 0)
	assert.NoError(t, err)
}

func subAccTestHelper(t *testing.T) string {
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
