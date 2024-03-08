package bitget

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
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
	exchCfg.Enabled = true

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
	_, err = bi.QueryAnnouncements(context.Background(), "latest_news", time.Time{}, time.Time{})
	assert.NoError(t, err)
}

func TestGetTime(t *testing.T) {
	_, err := bi.GetTime(context.Background())
	assert.NoError(t, err)
}

func TestGetTradeRate(t *testing.T) {
	_, err := bi.GetTradeRate(context.Background(), "BTCUSD", "spot")
	assert.NoError(t, err)
}
