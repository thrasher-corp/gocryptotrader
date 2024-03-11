package bitget

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	clientID                = "" // Passphrase made at API key creation
	canManipulateRealOrders = false
	testingInSandbox        = false
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
	_, err = bi.GetFuturesTransactionRecords(context.Background(), "COIN-FUTURES", "", time.Now().Add(-time.Hour*24*30), time.Now(), 0, 0)
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
	resp, err := bi.GetMerchantInfo(context.Background())
	assert.NoError(t, err)
	fmt.Print(resp)
}
