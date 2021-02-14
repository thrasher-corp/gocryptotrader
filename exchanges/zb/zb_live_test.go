//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package zb

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("ZB load config error", err)
	}
	zbConfig, err := cfg.GetExchangeConfig("ZB")
	if err != nil {
		log.Fatal("ZB Setup() init error", err)
	}
	zbConfig.API.AuthenticatedSupport = true
	zbConfig.API.Credentials.Key = apiKey
	zbConfig.API.Credentials.Secret = apiSecret
	z.SetDefaults()
	z.Websocket = sharedtestvalues.NewTestWebsocket()
	err = z.Setup(zbConfig)
	if err != nil {
		log.Fatal("ZB setup error", err)
	}
	log.Printf(sharedtestvalues.LiveTesting, z.Name)
	z.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	z.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	os.Exit(m.Run())
}
