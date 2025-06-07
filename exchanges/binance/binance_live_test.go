//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package binance

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = false

func TestMain(m *testing.M) {
	b = new(Binance)
	if err := testexch.Setup(b); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		b.API.AuthenticatedSupport = true
		b.API.CredentialsValidator.RequiresBase64DecodeSecret = false
		b.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}
	if useTestNet {
		for k, v := range map[exchange.URL]string{
			exchange.RestUSDTMargined: "https://testnet.binancefuture.com",
			exchange.RestCoinMargined: "https://testnet.binancefuture.com",
			exchange.RestSpot:         "https://testnet.binance.vision/api",
		} {
			if err := b.API.Endpoints.SetRunning(k.String(), v); err != nil {
				log.Fatalf("Testnet `%s` URL error with `%s`: %s", k, v, err)
			}
		}
	}
	b.setupOrderbookManager()
	b.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	log.Printf(sharedtestvalues.LiveTesting, b.Name)
	if err := b.populateTradablePairs(); err != nil {
		log.Fatal(err)
	}
	if err := b.populateTradablePairs(); err != nil {
		log.Fatal(err)
	}
	if mockTests {
		optionsTradablePair = currency.Pair{Base: currency.NewCode("ETH"), Quote: currency.NewCode("240927-3800-P"), Delimiter: currency.DashDelimiter}
		usdtmTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	}
	assetToTradablePairMap = map[asset.Item]currency.Pair{
		asset.Spot:                spotTradablePair,
		asset.Options:             optionsTradablePair,
		asset.USDTMarginedFutures: usdtmTradablePair,
		asset.CoinMarginedFutures: coinmTradablePair,
		asset.Margin:              spotTradablePair,
	}
	assetToTradablePairMap = map[asset.Item]currency.Pair{
		asset.Spot:                spotTradablePair,
		asset.Options:             optionsTradablePair,
		asset.USDTMarginedFutures: usdtmTradablePair,
		asset.CoinMarginedFutures: coinmTradablePair,
		asset.Margin:              spotTradablePair,
	}
	os.Exit(m.Run())
}
