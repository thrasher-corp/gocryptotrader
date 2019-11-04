package okgroup

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/websocket/wshandler"
)

const (
	apiKey    = ""
	apiSecret = ""

	testAPIURL     = "https://www.okex.com/api/"
	testAPIVersion = "/v3/"
)

var o OKGroup

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("okgroup load config error", err)
	}
	okgroup, err := cfg.GetExchangeConfig("Okex")
	if err != nil {
		log.Fatal("okgroup Setup() init error", err)
	}

	okgroup.API.AuthenticatedSupport = true
	okgroup.API.Credentials.Key = apiKey
	okgroup.API.Credentials.Secret = apiSecret
	o.API.Endpoints.URL = testAPIURL
	o.APIVersion = testAPIVersion

	o.Requester = request.New("okgroup_test_things",
		request.NewRateLimit(time.Second, 10),
		request.NewRateLimit(time.Second, 10),
		common.NewHTTPClientWithTimeout(exchange.DefaultHTTPTimeout),
	)
	o.Websocket = wshandler.New()

	err = o.Setup(okgroup)
	if err != nil {
		log.Fatal("okgroup setup error", err)
	}
	os.Exit(m.Run())
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := o.GetOrderBook(GetOrderBookRequest{InstrumentID: "BTC-USDT"},
		asset.Spot)
	if err != nil {
		t.Error(err)
	}

	// futures expire and break test, will need to mock this in the future
	_, err = o.GetOrderBook(GetOrderBookRequest{InstrumentID: "Payload"},
		asset.Futures)
	if err == nil {
		t.Error("error cannot be nil")
	}

	_, err = o.GetOrderBook(GetOrderBookRequest{InstrumentID: "BTC-USD-SWAP"},
		asset.PerpetualSwap)
	if err != nil {
		t.Error(err)
	}
}
