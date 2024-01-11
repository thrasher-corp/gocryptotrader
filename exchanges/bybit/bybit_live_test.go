//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bybit

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
)

var mockTests = false

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bybit")
	if err != nil {
		log.Fatal(err)
	}
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	request.MaxRequestJobs = 100
	err = b.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}
	err = instantiateTradablePairs()
	if err != nil {
		log.Fatalf("%s %v", b.Name, err)
	}
	err = b.RetrieveAndSetAccountType(context.Background())
	if err != nil {
		gctlog.Errorf(gctlog.ExchangeSys, "RetrieveAndSetAccountType: %v", err)
	}
	os.Exit(m.Run())
}

func instantiateTradablePairs() error {
	err := b.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		return err
	}
	tradables, err := b.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	format, err := b.GetPairFormat(asset.Spot, true)
	if err != nil {
		return err
	}
	spotTradablePair = tradables[0].Format(format)
	tradables, err = b.GetEnabledPairs(asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	format, err = b.GetPairFormat(asset.USDTMarginedFutures, true)
	if err != nil {
		return err
	}
	usdtMarginedTradablePair = tradables[0].Format(format)
	tradables, err = b.GetEnabledPairs(asset.USDCMarginedFutures)
	if err != nil {
		return err
	}
	format, err = b.GetPairFormat(asset.USDCMarginedFutures, true)
	if err != nil {
		return err
	}
	usdcMarginedTradablePair = tradables[0].Format(format)
	tradables, err = b.GetEnabledPairs(asset.CoinMarginedFutures)
	if err != nil {
		return err
	}
	format, err = b.GetPairFormat(asset.CoinMarginedFutures, true)
	if err != nil {
		return err
	}
	inverseTradablePair = tradables[0].Format(format)
	tradables, err = b.GetEnabledPairs(asset.Options)
	if err != nil {
		return err
	}
	format, err = b.GetPairFormat(asset.Options, true)
	if err != nil {
		return err
	}
	optionsTradablePair = tradables[0].Format(format)
	return nil
}
