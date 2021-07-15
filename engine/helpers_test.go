package engine

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/stats"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
)

var testExchange = "Bitstamp"

func CreateTestBot(t *testing.T) *Engine {
	cFormat := &currency.PairFormat{Uppercase: true}
	cp1 := currency.NewPair(currency.BTC, currency.USD)
	cp2 := currency.NewPair(currency.BTC, currency.USDT)

	pairs1 := map[asset.Item]*currency.PairStore{
		asset.Spot: {
			AssetEnabled: convert.BoolPtr(true),
			Available:    currency.Pairs{cp1},
			Enabled:      currency.Pairs{cp1},
		},
	}
	pairs2 := map[asset.Item]*currency.PairStore{
		asset.Spot: {
			AssetEnabled: convert.BoolPtr(true),
			Available:    currency.Pairs{cp2},
			Enabled:      currency.Pairs{cp2},
		},
	}
	bot := &Engine{
		ExchangeManager: SetupExchangeManager(),
		Config: &config.Config{Exchanges: []config.ExchangeConfig{
			{
				Name:                    testExchange,
				Enabled:                 true,
				WebsocketTrafficTimeout: time.Second,
				API: config.APIConfig{
					Credentials: config.APICredentialsConfig{},
				},
				CurrencyPairs: &currency.PairsManager{
					RequestFormat:   cFormat,
					ConfigFormat:    cFormat,
					UseGlobalFormat: true,
					Pairs:           pairs1,
				},
			},
			{
				Name:                    "binance",
				Enabled:                 true,
				WebsocketTrafficTimeout: time.Second,
				API: config.APIConfig{
					Credentials: config.APICredentialsConfig{},
				},
				CurrencyPairs: &currency.PairsManager{
					RequestFormat:   cFormat,
					ConfigFormat:    cFormat,
					UseGlobalFormat: true,
					Pairs:           pairs2,
				},
			},
		}}}
	err := bot.LoadExchange(testExchange, nil)
	if err != nil {
		t.Fatalf("SetupTest: Failed to load exchange: %s", err)
	}

	return bot
}

func TestGetExchangeOTPs(t *testing.T) {
	t.Parallel()
	bot := CreateTestBot(t)
	_, err := bot.GetExchangeOTPs()
	if err == nil {
		t.Fatal("Expected err with no exchange OTP secrets set")
	}

	bnCfg, err := bot.Config.GetExchangeConfig("binance")
	if err != nil {
		t.Fatal(err)
	}
	bCfg, err := bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Fatal(err)
	}

	bnCfg.API.Credentials.OTPSecret = "JBSWY3DPEHPK3PXP"
	bCfg.API.Credentials.OTPSecret = "JBSWY3DPEHPK3PXP"
	result, err := bot.GetExchangeOTPs()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 2 {
		t.Fatal("Expected 2 OTP results")
	}

	bnCfg.API.Credentials.OTPSecret = "°"
	result, err = bot.GetExchangeOTPs()
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1 {
		t.Fatal("Expected 1 OTP code with invalid OTP Secret")
	}

	// Flush settings
	bnCfg.API.Credentials.OTPSecret = ""
	bCfg.API.Credentials.OTPSecret = ""
}

func TestGetExchangeoOTPByName(t *testing.T) {
	t.Parallel()
	bot := CreateTestBot(t)
	_, err := bot.GetExchangeOTPByName(testExchange)
	if err == nil {
		t.Fatal("Expected err with no exchange OTP secrets set")
	}

	bCfg, err := bot.Config.GetExchangeConfig(testExchange)
	if err != nil {
		t.Fatal(err)
	}

	bCfg.API.Credentials.OTPSecret = "JBSWY3DPEHPK3PXP"
	result, err := bot.GetExchangeOTPByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	if result == "" {
		t.Fatal("Expected valid OTP code")
	}

	// Flush setting
	bCfg.API.Credentials.OTPSecret = ""
}

func TestGetAuthAPISupportedExchanges(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)
	if result := e.GetAuthAPISupportedExchanges(); len(result) != 0 {
		t.Fatal("Unexpected result", result)
	}

	exch := e.ExchangeManager.GetExchangeByName(testExchange)
	b := exch.GetBase()
	b.API.AuthenticatedWebsocketSupport = true
	b.API.Credentials.Key = "test"
	b.API.Credentials.Secret = "test"
	if result := e.GetAuthAPISupportedExchanges(); len(result) != 1 {
		t.Fatal("Unexpected result", result)
	}
}

func TestIsOnline(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)
	var err error
	e.connectionManager, err = setupConnectionManager(&e.Config.ConnectionMonitor)
	if err != nil {
		t.Fatal(err)
	}
	if r := e.IsOnline(); r {
		t.Fatal("Unexpected result")
	}

	if err = e.connectionManager.Start(); err != nil {
		t.Fatal(err)
	}

	tick := time.NewTicker(time.Second * 5)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			t.Fatal("Test timeout")
		default:
			if e.IsOnline() {
				if err := e.connectionManager.Stop(); err != nil {
					t.Fatal("unable to shutdown connection manager")
				}
				return
			}
		}
	}
}

func TestGetSpecificAvailablePairs(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)
	c := currency.Code{
		Item: &currency.Item{
			Role:   currency.Cryptocurrency,
			Symbol: "usdt",
		},
	}
	e.Config = &config.Config{
		Exchanges: []config.ExchangeConfig{
			{
				Enabled: true,
				Name:    testExchange,
				CurrencyPairs: &currency.PairsManager{Pairs: map[asset.Item]*currency.PairStore{
					asset.Spot: {
						AssetEnabled: convert.BoolPtr(true),
						Enabled:      currency.Pairs{currency.NewPair(currency.BTC, currency.USD), currency.NewPair(currency.BTC, c)},
						Available:    currency.Pairs{currency.NewPair(currency.BTC, currency.USD), currency.NewPair(currency.BTC, c)},
						ConfigFormat: &currency.PairFormat{
							Uppercase: true,
						},
					},
				}},
			},
		},
	}
	assetType := asset.Spot

	result := e.GetSpecificAvailablePairs(true, true, true, true, assetType)
	btcUSD := currency.NewPair(currency.BTC, currency.USD)
	if !result.Contains(btcUSD, true) {
		t.Error("Unexpected result")
	}

	btcUSDT := currency.NewPair(currency.BTC, c)
	if !result.Contains(btcUSDT, false) {
		t.Error("Unexpected result")
	}

	result = e.GetSpecificAvailablePairs(true, true, false, false, assetType)

	if result.Contains(btcUSDT, false) {
		t.Error("Unexpected result")
	}

	ltcBTC := currency.NewPair(currency.LTC, currency.BTC)
	result = e.GetSpecificAvailablePairs(true, false, false, true, assetType)
	if result.Contains(ltcBTC, false) {
		t.Error("Unexpected result")
	}
}

func TestIsRelatablePairs(t *testing.T) {
	t.Parallel()
	CreateTestBot(t)
	xbtusd, err := currency.NewPairFromStrings("XBT", "USD")
	if err != nil {
		t.Fatal(err)
	}

	btcusd, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	// Test relational pairs with similar names
	result := IsRelatablePairs(xbtusd, btcusd, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names reversed
	result = IsRelatablePairs(btcusd, xbtusd, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	btcusdt, err := currency.NewPairFromStrings("BTC", "USDT")
	if err != nil {
		t.Fatal(err)
	}

	// Test relational pairs with similar names but with Tether support disabled
	result = IsRelatablePairs(xbtusd, btcusdt, false)
	if result {
		t.Fatal("Unexpected result")
	}

	xbtusdt, err := currency.NewPairFromStrings("XBT", "USDT")
	if err != nil {
		t.Fatal(err)
	}

	// Test relational pairs with similar names but with Tether support enabled
	result = IsRelatablePairs(xbtusdt, btcusd, true)
	if !result {
		t.Fatal("Unexpected result")
	}

	aeusdt, err := currency.NewPairFromStrings("AE", "USDT")
	if err != nil {
		t.Fatal(err)
	}

	usdtae, err := currency.NewPairDelimiter("USDT-AE", "-")
	if err != nil {
		t.Fatal(err)
	}

	// Test relational pairs with different ordering, a delimiter and with
	// Tether support enabled
	result = IsRelatablePairs(aeusdt, usdtae, true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with different ordering, a delimiter and with
	// Tether support disabled
	result = IsRelatablePairs(aeusdt, usdtae, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	xbteur, err := currency.NewPairFromStrings("XBT", "EUR")
	if err != nil {
		t.Fatal(err)
	}

	btcaud, err := currency.NewPairFromStrings("BTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl pairs with similar names and different fiat currencies
	result = IsRelatablePairs(xbteur, btcaud, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	usdbtc, err := currency.NewPairFromStrings("USD", "BTC")
	if err != nil {
		t.Fatal(err)
	}

	btceur, err := currency.NewPairFromStrings("BTC", "EUR")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl pairs with similar names, different fiat currencies and
	// with different ordering
	result = IsRelatablePairs(usdbtc, btceur, false)
	if !result { // Is this really expected result???
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names, different fiat currencies and
	// with Tether enabled
	result = IsRelatablePairs(usdbtc, btcusdt, true)
	if !result {
		t.Fatal("Unexpected result")
	}

	ltcbtc, err := currency.NewPairFromStrings("LTC", "BTC")
	if err != nil {
		t.Fatal(err)
	}

	btcltc, err := currency.NewPairFromStrings("BTC", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl crypto pairs with similar names
	result = IsRelatablePairs(ltcbtc, btcltc, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	ltceth, err := currency.NewPairFromStrings("LTC", "ETH")
	if err != nil {
		t.Fatal(err)
	}

	btceth, err := currency.NewPairFromStrings("BTC", "ETH")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl crypto pairs with similar different pairs
	result = IsRelatablePairs(ltceth, btceth, false)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar different pairs and with USDT
	// enabled
	usdtusd, err := currency.NewPairFromStrings("USDT", "USD")
	if err != nil {
		t.Fatal(err)
	}

	result = IsRelatablePairs(usdtusd, btcusd, true)
	if result {
		t.Fatal("Unexpected result")
	}

	xbtltc, err := currency.NewPairFromStrings("XBT", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl crypto pairs with with similar names
	result = IsRelatablePairs(xbtltc, btcltc, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	ltcxbt, err := currency.NewPairFromStrings("LTC", "XBT")
	if err != nil {
		t.Fatal(err)
	}

	// Test relationl crypto pairs with different ordering and similar names
	result = IsRelatablePairs(ltcxbt, btcltc, false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test edge case between two pairs when currency translations were causing
	// non-relational pairs to be relatable
	eurusd, err := currency.NewPairFromStrings("EUR", "USD")
	if err != nil {
		t.Fatal(err)
	}

	result = IsRelatablePairs(eurusd, btcusd, false)
	if result {
		t.Fatal("Unexpected result")
	}
}

func TestGetRelatableCryptocurrencies(t *testing.T) {
	t.Parallel()
	CreateTestBot(t)
	btcltc, err := currency.NewPairFromStrings("BTC", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	btcbtc, err := currency.NewPairFromStrings("BTC", "BTC")
	if err != nil {
		t.Fatal(err)
	}

	ltcltc, err := currency.NewPairFromStrings("LTC", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	btceth, err := currency.NewPairFromStrings("BTC", "ETH")
	if err != nil {
		t.Fatal(err)
	}

	p := GetRelatableCryptocurrencies(btcltc)
	if p.Contains(btcltc, true) {
		t.Error("Unexpected result")
	}
	if p.Contains(btcbtc, true) {
		t.Error("Unexpected result")
	}
	if p.Contains(ltcltc, true) {
		t.Error("Unexpected result")
	}
	if !p.Contains(btceth, true) {
		t.Error("Unexpected result")
	}

	p = GetRelatableCryptocurrencies(btcltc)
	if p.Contains(btcltc, true) {
		t.Error("Unexpected result")
	}
	if p.Contains(btcbtc, true) {
		t.Error("Unexpected result")
	}
	if p.Contains(ltcltc, true) {
		t.Error("Unexpected result")
	}
	if !p.Contains(btceth, true) {
		t.Error("Unexpected result")
	}
}

func TestGetRelatableFiatCurrencies(t *testing.T) {
	t.Parallel()
	btcUSD, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	btcEUR, err := currency.NewPairFromStrings("BTC", "EUR")
	if err != nil {
		t.Fatal(err)
	}

	p := GetRelatableFiatCurrencies(btcUSD)
	if !p.Contains(btcEUR, true) {
		t.Error("Unexpected result")
	}

	if p.Contains(currency.NewPair(currency.DOGE, currency.XRP), true) {
		t.Error("Unexpected result")
	}
}

func TestMapCurrenciesByExchange(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)

	var pairs = []currency.Pair{
		currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.BTC, currency.EUR),
	}

	result := e.MapCurrenciesByExchange(pairs, true, asset.Spot)
	pairs, ok := result[testExchange]
	if !ok {
		t.Fatal("Unexpected result")
	}

	if len(pairs) != 2 {
		t.Fatal("Unexpected result")
	}
}

func TestGetExchangeNamesByCurrency(t *testing.T) {
	t.Parallel()
	btsusd, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	btcjpy, err := currency.NewPairFromStrings("BTC", "JPY")
	if err != nil {
		t.Fatal(err)
	}

	blahjpy, err := currency.NewPairFromStrings("blah", "JPY")
	if err != nil {
		t.Fatal(err)
	}

	e := CreateTestBot(t)
	bf := "Bitflyer"
	e.Config.Exchanges = append(e.Config.Exchanges, config.ExchangeConfig{
		Enabled: true,
		Name:    bf,
		CurrencyPairs: &currency.PairsManager{Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled: convert.BoolPtr(true),
				Enabled:      currency.Pairs{btcjpy},
				Available:    currency.Pairs{btcjpy},
				ConfigFormat: &currency.PairFormat{
					Uppercase: true,
				},
			},
		}},
	})
	assetType := asset.Spot

	result := e.GetExchangeNamesByCurrency(btsusd,
		true,
		assetType)
	if !common.StringDataCompare(result, testExchange) {
		t.Fatal("Unexpected result")
	}

	result = e.GetExchangeNamesByCurrency(btcjpy,
		true,
		assetType)
	if !common.StringDataCompare(result, bf) {
		t.Fatal("Unexpected result")
	}

	result = e.GetExchangeNamesByCurrency(blahjpy,
		true,
		assetType)
	if len(result) > 0 {
		t.Fatal("Unexpected result")
	}
}

func TestGetSpecificOrderbook(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)

	var bids []orderbook.Item
	bids = append(bids, orderbook.Item{Price: 1000, Amount: 1})

	base := orderbook.Base{
		Pair:     currency.NewPair(currency.BTC, currency.USD),
		Bids:     bids,
		Exchange: "Bitstamp",
		Asset:    asset.Spot,
	}

	err := base.Process()
	if err != nil {
		t.Fatal("Unexpected result", err)
	}

	btsusd, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	ob, err := e.GetSpecificOrderbook(btsusd, testExchange, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if ob.Bids[0].Price != 1000 {
		t.Fatal("Unexpected result")
	}

	ethltc, err := currency.NewPairFromStrings("ETH", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	_, err = e.GetSpecificOrderbook(ethltc, testExchange, asset.Spot)
	if err == nil {
		t.Fatal("Unexpected result")
	}

	err = e.UnloadExchange(testExchange)
	if err != nil {
		t.Error(err)
	}
}

func TestGetSpecificTicker(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)
	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	err = ticker.ProcessTicker(&ticker.Price{
		Pair:         p,
		Last:         1000,
		AssetType:    asset.Spot,
		ExchangeName: testExchange})
	if err != nil {
		t.Fatal("ProcessTicker error", err)
	}

	tick, err := e.GetSpecificTicker(p, testExchange, asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if tick.Last != 1000 {
		t.Fatal("Unexpected result")
	}

	ethltc, err := currency.NewPairFromStrings("ETH", "LTC")
	if err != nil {
		t.Fatal(err)
	}

	_, err = e.GetSpecificTicker(ethltc, testExchange, asset.Spot)
	if err == nil {
		t.Fatal("Unexpected result")
	}

	err = e.UnloadExchange(testExchange)
	if err != nil {
		t.Error(err)
	}
}

func TestGetCollatedExchangeAccountInfoByCoin(t *testing.T) {
	t.Parallel()
	CreateTestBot(t)

	var exchangeInfo []account.Holdings

	var bitfinexHoldings account.Holdings
	bitfinexHoldings.Exchange = "Bitfinex"
	bitfinexHoldings.Accounts = append(bitfinexHoldings.Accounts,
		account.SubAccount{
			Currencies: []account.Balance{
				{
					CurrencyName: currency.BTC,
					TotalValue:   100,
					Hold:         0,
				},
			},
		})

	exchangeInfo = append(exchangeInfo, bitfinexHoldings)

	var bitstampHoldings account.Holdings
	bitstampHoldings.Exchange = testExchange
	bitstampHoldings.Accounts = append(bitstampHoldings.Accounts,
		account.SubAccount{
			Currencies: []account.Balance{
				{
					CurrencyName: currency.LTC,
					TotalValue:   100,
					Hold:         0,
				},
				{
					CurrencyName: currency.BTC,
					TotalValue:   100,
					Hold:         0,
				},
			},
		})

	exchangeInfo = append(exchangeInfo, bitstampHoldings)

	result := GetCollatedExchangeAccountInfoByCoin(exchangeInfo)
	if len(result) == 0 {
		t.Fatal("Unexpected result")
	}

	amount, ok := result[currency.BTC]
	if !ok {
		t.Fatal("Expected currency was not found in result map")
	}

	if amount.TotalValue != 200 {
		t.Fatal("Unexpected result")
	}

	_, ok = result[currency.ETH]
	if ok {
		t.Fatal("Unexpected result")
	}
}

func TestGetExchangeHighestPriceByCurrencyPair(t *testing.T) {
	t.Parallel()
	CreateTestBot(t)

	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	err = stats.Add("Bitfinex", p, asset.Spot, 1000, 10000)
	if err != nil {
		t.Error(err)
	}
	err = stats.Add(testExchange, p, asset.Spot, 1337, 10000)
	if err != nil {
		t.Error(err)
	}
	exchangeName, err := GetExchangeHighestPriceByCurrencyPair(p, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if exchangeName != testExchange {
		t.Error("Unexpected result")
	}

	btcaud, err := currency.NewPairFromStrings("BTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetExchangeHighestPriceByCurrencyPair(btcaud, asset.Spot)
	if err == nil {
		t.Error("Unexpected result")
	}
}

func TestGetExchangeLowestPriceByCurrencyPair(t *testing.T) {
	t.Parallel()
	CreateTestBot(t)

	p, err := currency.NewPairFromStrings("BTC", "USD")
	if err != nil {
		t.Fatal(err)
	}

	err = stats.Add("Bitfinex", p, asset.Spot, 1000, 10000)
	if err != nil {
		t.Error(err)
	}
	err = stats.Add(testExchange, p, asset.Spot, 1337, 10000)
	if err != nil {
		t.Error(err)
	}
	exchangeName, err := GetExchangeLowestPriceByCurrencyPair(p, asset.Spot)
	if err != nil {
		t.Error(err)
	}

	if exchangeName != "Bitfinex" {
		t.Error("Unexpected result")
	}

	btcaud, err := currency.NewPairFromStrings("BTC", "AUD")
	if err != nil {
		t.Fatal(err)
	}

	_, err = GetExchangeLowestPriceByCurrencyPair(btcaud, asset.Spot)
	if err == nil {
		t.Error("Unexpected reuslt")
	}
}

func TestGetCryptocurrenciesByExchange(t *testing.T) {
	t.Parallel()
	e := CreateTestBot(t)

	_, err := e.GetCryptocurrenciesByExchange("Bitfinex", false, false, asset.Spot)
	if err != nil {
		t.Fatalf("Err %s", err)
	}
}

func TestGetExchangeNames(t *testing.T) {
	t.Parallel()
	bot := CreateTestBot(t)
	if e := bot.GetExchangeNames(true); len(e) == 0 {
		t.Error("exchange names should be populated")
	}
	if err := bot.UnloadExchange(testExchange); err != nil {
		t.Fatal(err)
	}
	if e := bot.GetExchangeNames(true); common.StringDataCompare(e, testExchange) {
		t.Error("Bitstamp should be missing")
	}
	if e := bot.GetExchangeNames(false); len(e) != 0 {
		t.Errorf("Expected %v Received %v", len(e), 0)
	}

	for i := range bot.Config.Exchanges {
		exch, err := bot.ExchangeManager.NewExchangeByName(bot.Config.Exchanges[i].Name)
		if err != nil && !errors.Is(err, ErrExchangeAlreadyLoaded) {
			t.Fatal(err)
		}
		if exch != nil {
			exch.SetDefaults()
			bot.ExchangeManager.Add(exch)
		}
	}
	if e := bot.GetExchangeNames(false); len(e) != len(bot.Config.Exchanges) {
		t.Errorf("Expected %v Received %v", len(bot.Config.Exchanges), len(e))
	}
}

func mockCert(derType string, notAfter time.Time) ([]byte, error) {
	privKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, err
	}

	host, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	dnsNames := []string{host}
	if host != "localhost" {
		dnsNames = append(dnsNames, "localhost")
	}

	if notAfter.IsZero() {
		notAfter = time.Now().Add(time.Hour * 24 * 365)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"gocryptotrader"},
			CommonName:   host,
		},
		NotBefore:             time.Now(),
		NotAfter:              notAfter,
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
		DNSNames: dnsNames,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		return nil, err
	}

	if derType == "" {
		derType = "CERTIFICATE"
	}

	certData := pem.EncodeToMemory(&pem.Block{Type: derType, Bytes: derBytes})
	if certData == nil {
		return nil, err
	}

	return certData, nil
}

func TestVerifyCert(t *testing.T) {
	t.Parallel()

	tester := []struct {
		PEMType       string
		CreateBypass  bool
		NotAfter      time.Time
		ErrorExpected error
	}{
		{
			ErrorExpected: nil,
		},
		{
			CreateBypass:  true,
			ErrorExpected: errCertDataIsNil,
		},
		{
			PEMType:       "MEOW",
			ErrorExpected: errCertTypeInvalid,
		},
		{
			NotAfter:      time.Now().Add(-time.Hour),
			ErrorExpected: errCertExpired,
		},
	}

	for x := range tester {
		var cert []byte
		var err error
		if !tester[x].CreateBypass {
			cert, err = mockCert(tester[x].PEMType, tester[x].NotAfter)
			if err != nil {
				t.Errorf("test %d unexpected error: %s", x, err)
				continue
			}
		}
		err = verifyCert(cert)
		if err != tester[x].ErrorExpected {
			t.Fatalf("test %d expected %v, got %v", x, tester[x].ErrorExpected, err)
		}
	}
}

func TestCheckAndGenCerts(t *testing.T) {
	t.Parallel()

	tempDir := filepath.Join(os.TempDir(), "gct-temp-tls")
	cleanup := func() {
		if err := os.RemoveAll(tempDir); err != nil {
			t.Errorf("unable to remove temp dir %s, manual deletion required", tempDir)
		}
	}

	if err := genCert(tempDir); err != nil {
		cleanup()
		t.Fatal(err)
	}

	defer cleanup()
	if err := checkCerts(tempDir); err != nil {
		t.Fatal(err)
	}

	// Now delete cert.pem and test regeneration of cert/key files
	certFile := filepath.Join(tempDir, "cert.pem")
	if err := os.Remove(certFile); err != nil {
		t.Fatal(err)
	}
	if err := checkCerts(tempDir); err != nil {
		t.Fatal(err)
	}

	// Now call checkCerts to test an expired cert
	certData, err := mockCert("", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	err = file.Write(certFile, certData)
	if err != nil {
		t.Fatal(err)
	}
	if err = checkCerts(tempDir); err != nil {
		t.Fatal(err)
	}
}
