//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package binance

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

var mockTests = false

func TestMain(m *testing.M) {
	if livetest.ShouldSkip() {
		log.Printf(livetest.LiveTestingSkipped, "Binance")
		os.Exit(0)
	}

	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Binance Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.CredentialsValidator.RequiresBase64DecodeSecret = false
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}
	if useTestNet {
		for k, v := range map[exchange.URL]string{
			exchange.RestUSDTMargined: "https://testnet.binancefuture.com",
			exchange.RestCoinMargined: "https://testnet.binancefuture.com",
			exchange.RestSpot:         "https://testnet.binance.vision/api",
		} {
			if err := e.API.Endpoints.SetRunningURL(k.String(), v); err != nil {
				log.Fatalf("Binance SetRunningURL error: %s", err)
			}
		}
	}
	e.setupOrderbookManager(context.Background())
	e.Websocket.DataHandler = stream.NewRelay(sharedtestvalues.WebsocketRelayBufferCapacity)
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	if err := e.populateTradablePairs(); err != nil {
		log.Fatal(err)
	}
	spotTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	marginTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	usdtmTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	coinmTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USD_PERP"))
	optionsTradablePair = currency.Pair{Base: currency.NewCode("BTC"), Quote: currency.NewCode("260327-100000-C"), Delimiter: currency.DashDelimiter}

	assetToTradablePairMap = map[asset.Item]currency.Pair{
		asset.Spot:                spotTradablePair,
		asset.Margin:              marginTradablePair,
		asset.Options:             optionsTradablePair,
		asset.USDTMarginedFutures: usdtmTradablePair,
		asset.CoinMarginedFutures: coinmTradablePair,
	}
	os.Exit(m.Run())
}

func (e *Exchange) populateTradablePairs() error {
	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		return err
	}
	tradablePairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w for %v", currency.ErrCurrencyPairsEmpty, asset.Spot)
	}
	spotTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Spot)
	if err != nil {
		return err
	}
	tradablePairs, err = e.GetEnabledPairs(asset.Margin)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w for %v", currency.ErrCurrencyPairsEmpty, asset.Margin)
	}
	marginTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Margin)
	if err != nil {
		return err
	}
	tradablePairs, err = e.GetEnabledPairs(asset.USDTMarginedFutures)
	if err != nil {
		return err
	}
	if len(tradablePairs) != 0 {
		usdtmTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.USDTMarginedFutures)
		if err != nil {
			return err
		}
	}
	tradablePairs, err = e.GetEnabledPairs(asset.CoinMarginedFutures)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		coinmTradablePair, err = currency.NewPairFromString("ETHUSD_PERP")
		if err != nil {
			return err
		}
	} else {
		coinmTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.CoinMarginedFutures)
		if err != nil {
			return err
		}
	}
	tradablePairs, err = e.GetEnabledPairs(asset.Options)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return fmt.Errorf("%w for %v", currency.ErrCurrencyPairsEmpty, asset.Options)
	}
	optionsTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Options)
	e.HTTPRecording = true
	return err
}
