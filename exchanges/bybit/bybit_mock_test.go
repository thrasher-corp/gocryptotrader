//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bybit

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const mockfile = "../../testdata/http_mock/bybit/bybit.json"

var mockTests = true

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	bybitConfig, err := cfg.GetExchangeConfig("Bybit")
	if err != nil {
		log.Fatal("Bybit Setup() init error", err)
	}

	b.SkipAuthCheck = true
	request.MaxRequestJobs = 100
	bybitConfig.API.Credentials.Key = apiKey
	bybitConfig.API.Credentials.Secret = apiSecret
	bybitConfig.API.AuthenticatedSupport = true
	bybitConfig.API.AuthenticatedWebsocketSupport = true
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	err = b.Setup(bybitConfig)
	if err != nil {
		log.Fatal("Bybit setup error", err)
	}

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}
	err = b.SetHTTPClient(newClient)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}
	endpointMap := b.API.Endpoints.GetURLMap()
	for k := range endpointMap {
		err = b.API.Endpoints.SetRunning(k, serverDetails)
		if err != nil {
			log.Fatal(err)
		}
	}
	request.MaxRequestJobs = 100
	err = b.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		log.Fatal("Bybit setup error", err)
	}
	spotTradablePair = currency.Pair{Base: currency.BTC, Quote: currency.USDT}
	okay, err := b.IsPairEnabled(spotTradablePair, asset.Spot)
	if !okay || err != nil {
		err = b.CurrencyPairs.EnablePair(asset.Spot, spotTradablePair)
		if err != nil {
			log.Fatal(err)
		}
	}
	usdtMarginedTradablePair = currency.Pair{Base: currency.NewCode("10000LADYS"), Quote: currency.USDT}
	if okay, err = b.IsPairEnabled(usdtMarginedTradablePair, asset.USDTMarginedFutures); !okay || err != nil {
		err = b.CurrencyPairs.EnablePair(asset.USDTMarginedFutures, usdtMarginedTradablePair)
		if err != nil {
			log.Fatal(err)
		}
	}
	usdcMarginedTradablePair = currency.Pair{Base: currency.ETH, Quote: currency.PERP}
	if okay, err = b.IsPairEnabled(usdcMarginedTradablePair, asset.USDCMarginedFutures); !okay || err != nil {
		err = b.CurrencyPairs.EnablePair(asset.USDCMarginedFutures, usdcMarginedTradablePair)
		if err != nil {
			log.Fatal(err)
		}
	}
	inverseTradablePair = currency.Pair{Base: currency.ADA, Quote: currency.USD}
	if okay, err = b.IsPairEnabled(inverseTradablePair, asset.CoinMarginedFutures); !okay || err != nil {
		err = b.CurrencyPairs.EnablePair(asset.CoinMarginedFutures, inverseTradablePair)
		if err != nil {
			log.Fatal(err)
		}
	}
	optionsTradablePair = currency.Pair{Base: currency.BTC, Delimiter: currency.DashDelimiter, Quote: currency.NewCode("29DEC23-80000-C")}
	if okay, err = b.IsPairEnabled(optionsTradablePair, asset.Options); !okay || err != nil {
		err = b.CurrencyPairs.EnablePair(asset.Options, optionsTradablePair)
		if err != nil {
			log.Fatal(err)
		}
	}
	os.Exit(m.Run())
}
