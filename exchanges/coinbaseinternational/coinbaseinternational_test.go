package coinbaseinternational

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	passphrase              = ""
	canManipulateRealOrders = false
)

var co = &CoinbaseInternational{}

func TestMain(m *testing.M) {
	co.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Coinbaseinternational")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.Enabled = true
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.Credentials.ClientID = passphrase
	co.Websocket = sharedtestvalues.NewTestWebsocket()
	err = co.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(CoinbaseInternational); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

// Implement tests for API endpoints below

func TestListAssets(t *testing.T) {
	t.Parallel()
	_, err := co.ListAssets(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetAssetDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetAssetDetails(context.Background(), currency.EMPTYCODE, "", "207597618027560960")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSupportedNetworksPerAsset(t *testing.T) {
	t.Parallel()
	_, err := co.GetSupportedNetworksPerAsset(context.Background(), currency.BTC, "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstruments(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetInstrumentDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetInstrumentDetails(context.Background(), "BTC-PERP", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetQuotePerInstrument(t *testing.T) {
	t.Parallel()
	_, err := co.GetQuotePerInstrument(context.Background(), "BTC-PERP", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestCreateOrder(t *testing.T) {
	t.Parallel()
	orderType, err := orderTypeString(order.Limit)
	if err != nil {
		t.Fatal(err)
	}
	co.Verbose = true
	_, err = co.CreateOrder(context.Background(), &OrderRequestParams{
		Side:       "BUY",
		BaseSize:   1,
		Instrument: "BTC-USDT",
		OrderType:  orderType,
		Price:      12345.67,
		ExpireTime: "",
		PostOnly:   true,
	})
	if err != nil {
		t.Error(err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	// sharedtestvalues.SkipTestIfCredentialsUnset(t, )
	_, err := co.GetOpenOrders(context.Background(), "", "", "BTC-PERP", "", "", time.Time{}, 0, 0)
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := co.CancelOrders(context.Background(), "1234", "", "")
	if err != nil {
		t.Error(err)
	}
}

func TestModifyOpenOrder(t *testing.T) {
	t.Parallel()
	_, err := co.ModifyOpenOrder(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestGetOrderDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetOrderDetails(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := co.CancelTradeOrder(context.Background(), "1234")
	if err != nil {
		t.Error(err)
	}
}

func TestListAllUserPortfolios(t *testing.T) {
	t.Parallel()
	_, err := co.GetAllUserPortfolios(context.Background())
	if err != nil {
		t.Error(err)
	}
}

func TestGetPortfolioDetails(t *testing.T) {
	t.Parallel()
	_, err := co.GetPortfolioDetails(context.Background(), "", "1234")
	if err != nil {
		t.Error(err)
	}
}
