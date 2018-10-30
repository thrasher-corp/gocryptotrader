package engine

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	TestConfig = "../testdata/configtest.json"
)

var (
	helperTestLoaded = false
)

func SetupTestHelpers(t *testing.T) {
	if !helperTestLoaded {
		if !testSetup {
			if Bot == nil {
				Bot = new(Engine)
			}
			Bot.Config = &config.Cfg
			err := Bot.Config.LoadConfig("../testdata/configtest.json")
			if err != nil {
				t.Fatalf("Test failed. SetupTest: Failed to load config: %s", err)
			}
			testSetup = true
		}
		err := Bot.Config.RetrieveConfigCurrencyPairs(true)
		if err != nil {
			t.Fatalf("Failed to retrieve config currency pairs. %s", err)
		}
		helperTestLoaded = true
	}
}

func TestGetSpecificAvailablePairs(t *testing.T) {
	SetupTestHelpers(t)
	assetType := assets.AssetTypeSpot
	result := GetSpecificAvailablePairs(true, true, true, false, assetType)

	if !result.Contains(currency.NewPairFromStrings("BTC", "USD"), true) {
		t.Fatal("Unexpected result")
	}

	if !result.Contains(currency.NewPairFromStrings("BTC", "USDT"), false) {
		t.Fatal("Unexpected result")
	}

	result = GetSpecificAvailablePairs(true, true, false, false, assetType)

	if result.Contains(currency.NewPairFromStrings("BTC", "USDT"), false) {
		t.Fatal("Unexpected result")
	}

	result = GetSpecificAvailablePairs(true, false, false, true, assetType)
	if !result.Contains(currency.NewPairFromStrings("LTC", "BTC"), false) {
		t.Fatal("Unexpected result")
	}
}

func TestIsRelatablePairs(t *testing.T) {
	SetupTestHelpers(t)

	// Test relational pairs with similar names
	result := IsRelatablePairs(currency.NewPairFromStrings("XBT", "USD"),
		currency.NewPairFromStrings("BTC", "USD"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names reversed
	result = IsRelatablePairs(currency.NewPairFromStrings("BTC", "USD"),
		currency.NewPairFromStrings("XBT", "USD"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names but with Tether support disabled
	result = IsRelatablePairs(currency.NewPairFromStrings("XBT", "USD"),
		currency.NewPairFromStrings("BTC", "USDT"), false)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names but with Tether support enabled
	result = IsRelatablePairs(currency.NewPairFromStrings("XBT", "USDT"),
		currency.NewPairFromStrings("BTC", "USD"), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with different ordering, a delimiter and with
	// Tether support enabled
	result = IsRelatablePairs(currency.NewPairFromStrings("AE", "USDT"),
		currency.NewPairDelimiter("USDT-AE", "-"), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with different ordering, a delimiter and with
	// Tether support disabled
	result = IsRelatablePairs(currency.NewPairFromStrings("AE", "USDT"),
		currency.NewPairDelimiter("USDT-AE", "-"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names and different fiat currencies
	result = IsRelatablePairs(currency.NewPairFromStrings("XBT", "EUR"),
		currency.NewPairFromStrings("BTC", "AUD"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names, different fiat currencies and
	// with different ordering
	result = IsRelatablePairs(currency.NewPairFromStrings("USD", "BTC"),
		currency.NewPairFromStrings("BTC", "EUR"), false)
	if !result { // Is this really expected result???
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names, different fiat currencies and
	// with Tether enabled
	result = IsRelatablePairs(currency.NewPairFromStrings("USD", "BTC"),
		currency.NewPairFromStrings("BTC", "USDT"), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar names
	result = IsRelatablePairs(currency.NewPairFromStrings("LTC", "BTC"),
		currency.NewPairFromStrings("BTC", "LTC"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar different pairs
	result = IsRelatablePairs(currency.NewPairFromStrings("LTC", "ETH"),
		currency.NewPairFromStrings("BTC", "ETH"), false)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar different pairs and with USDT
	// enabled
	result = IsRelatablePairs(currency.NewPairFromStrings("USDT", "USD"),
		currency.NewPairFromStrings("BTC", "USD"), true)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with with similar names
	result = IsRelatablePairs(currency.NewPairFromStrings("XBT", "LTC"),
		currency.NewPairFromStrings("BTC", "LTC"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with different ordering and similar names
	result = IsRelatablePairs(currency.NewPairFromStrings("LTC", "XBT"),
		currency.NewPairFromStrings("BTC", "LTC"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test edge case between two pairs when currency translations were causing
	// non-relational pairs to be relatable
	result = IsRelatablePairs(currency.NewPairFromStrings("EUR", "USD"),
		currency.NewPairFromStrings("BTC", "USD"), false)
	if result {
		t.Fatal("Unexpected result")
	}
}

func TestGetRelatableCryptocurrencies(t *testing.T) {
	SetupTestHelpers(t)
	p := GetRelatableCryptocurrencies(currency.NewPairFromStrings("BTC", "LTC"))
	if p.Contains(currency.NewPairFromStrings("BTC", "LTC"), true) {
		t.Fatal("Unexpected result")
	}
	if p.Contains(currency.NewPairFromStrings("BTC", "BTC"), true) {
		t.Fatal("Unexpected result")
	}
	if p.Contains(currency.NewPairFromStrings("LTC", "LTC"), true) {
		t.Fatal("Unexpected result")
	}
	if !p.Contains(currency.NewPairFromStrings("BTC", "ETH"), true) {
		t.Fatal("Unexpected result")
	}

	p = GetRelatableCryptocurrencies(currency.NewPairFromStrings("BTC", "LTC"))
	if p.Contains(currency.NewPairFromStrings("BTC", "LTC"), true) {
		t.Fatal("Unexpected result")
	}
	if p.Contains(currency.NewPairFromStrings("BTC", "BTC"), true) {
		t.Fatal("Unexpected result")
	}
	if p.Contains(currency.NewPairFromStrings("LTC", "LTC"), true) {
		t.Fatal("Unexpected result")
	}
	if !p.Contains(currency.NewPairFromStrings("BTC", "ETH"), true) {
		t.Fatal("Unexpected result")
	}
}

func TestGetRelatableFiatCurrencies(t *testing.T) {
	SetupTestHelpers(t)
	p := GetRelatableFiatCurrencies(currency.NewPairFromStrings("BTC", "USD"))
	if !p.Contains(currency.NewPairFromStrings("BTC", "EUR"), true) {
		t.Fatal("Unexpected result")
	}

	p = GetRelatableFiatCurrencies(currency.NewPairFromStrings("BTC", "USD"))
	if !p.Contains(currency.NewPairFromStrings("BTC", "ZAR"), true) {
		t.Fatal("Unexpected result")
	}
}

func TestMapCurrenciesByExchange(t *testing.T) {
	SetupTestHelpers(t)

	var pairs = []currency.Pair{
		currency.NewPair(currency.BTC, currency.USD),
		currency.NewPair(currency.BTC, currency.EUR),
	}

	result := MapCurrenciesByExchange(pairs, true, assets.AssetTypeSpot)
	pairs, ok := result["Bitstamp"]
	if !ok {
		t.Fatal("Unexpected result")
	}

	if len(pairs) != 2 {
		t.Fatal("Unexpected result")
	}
}

func TestGetExchangeNamesByCurrency(t *testing.T) {
	SetupTestHelpers(t)
	assetType := assets.AssetTypeSpot

	result := GetExchangeNamesByCurrency(currency.NewPairFromStrings("BTC", "USD"), true, assetType)
	if !common.StringDataCompare(result, "Bitstamp") {
		t.Fatal("Unexpected result")
	}

	result = GetExchangeNamesByCurrency(currency.NewPairFromStrings("BTC", "JPY"), true, assetType)
	t.Log(result)
	if !common.StringDataCompare(result, "Bitflyer") {
		t.Fatal("Unexpected result")
	}

	result = GetExchangeNamesByCurrency(currency.NewPairFromStrings("blah", "JPY"), true, assetType)
	if len(result) > 0 {
		t.Fatal("Unexpected result")
	}
}

func TestGetSpecificOrderbook(t *testing.T) {
	SetupTestHelpers(t)

	LoadExchange("Bitstamp", false, nil)

	bids := []orderbook.Item{}
	bids = append(bids, orderbook.Item{Price: 1000, Amount: 1})
	asset := assets.AssetTypeSpot

	base := orderbook.Base{
		Pair:         currency.NewPair(currency.BTC, currency.USD),
		Bids:         bids,
		ExchangeName: "Bitstamp",
		AssetType:    asset,
	}

	err := base.Process()
	if err != nil {
		t.Fatal("Unexpected result", err)
	}

	ob, err := GetSpecificOrderbook(currency.NewPairFromString("BTCUSD"), "Bitstamp", assets.AssetTypeSpot)
	if err != nil {
		t.Fatal(err)
	}

	if ob.Bids[0].Price != 1000 {
		t.Fatal("Unexpected result")
	}

	ob, err = GetSpecificOrderbook(currency.NewPairFromStrings("ETH", "LTC"), "Bitstamp", asset)
	if err == nil {
		t.Fatal("Unexpected result")
	}

	UnloadExchange("Bitstamp")
}

func TestGetSpecificTicker(t *testing.T) {
	SetupTestHelpers(t)

	LoadExchange("Bitstamp", false, nil)
	p := currency.NewPairFromStrings("BTC", "USD")
	asset := assets.AssetTypeSpot
	err := ticker.ProcessTicker("Bitstamp",
		&ticker.Price{Pair: p, Last: 1000},
		assets.AssetTypeSpot)
	if err != nil {
		t.Fatal("Test failed. ProcessTicker error", err)
	}

	tick, err := GetSpecificTicker(currency.NewPairFromStrings("BTC", "USD"), "Bitstamp", asset)
	if err != nil {
		t.Fatal(err)
	}

	if tick.Last != 1000 {
		t.Fatal("Unexpected result")
	}

	tick, err = GetSpecificTicker(currency.NewPairFromStrings("ETH", "LTC"), "Bitstamp", asset)
	if err == nil {
		t.Fatal("Unexpected result")
	}

	UnloadExchange("Bitstamp")
}

func TestGetCollatedExchangeAccountInfoByCoin(t *testing.T) {
	SetupTestHelpers(t)

	exchangeInfo := []exchange.AccountInfo{}
	var info exchange.AccountInfo

	info.Exchange = "Bitfinex"
	info.Accounts = append(info.Accounts,
		exchange.Account{
			Currencies: []exchange.AccountCurrencyInfo{
				{
					CurrencyName: currency.BTC,
					TotalValue:   100,
					Hold:         0,
				},
			},
		})

	exchangeInfo = append(exchangeInfo, info)

	info.Exchange = "Bitstamp"
	info.Accounts = append(info.Accounts,
		exchange.Account{
			Currencies: []exchange.AccountCurrencyInfo{
				{
					CurrencyName: currency.LTC,
					TotalValue:   100,
					Hold:         0,
				},
			},
		})

	exchangeInfo = append(exchangeInfo, info)

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

func TestGetAccountCurrencyInfoByExchangeName(t *testing.T) {
	SetupTestHelpers(t)

	exchangeInfo := []exchange.AccountInfo{}
	var info exchange.AccountInfo
	info.Exchange = "Bitfinex"
	info.Accounts = append(info.Accounts,
		exchange.Account{
			Currencies: []exchange.AccountCurrencyInfo{
				{
					CurrencyName: currency.BTC,
					TotalValue:   100,
					Hold:         0,
				},
			},
		})

	exchangeInfo = append(exchangeInfo, info)

	result, err := GetAccountCurrencyInfoByExchangeName(exchangeInfo, "Bitfinex")
	if err != nil {
		t.Fatal(err)
	}

	if result.Exchange != "Bitfinex" {
		t.Fatal("Unexepcted result")
	}

	_, err = GetAccountCurrencyInfoByExchangeName(exchangeInfo, "ASDF")
	if err != ErrExchangeNotFound {
		t.Fatal("Unexepcted result")
	}
}

func TestGetExchangeHighestPriceByCurrencyPair(t *testing.T) {
	SetupTestHelpers(t)

	p := currency.NewPairFromStrings("BTC", "USD")
	asset := assets.AssetTypeSpot
	stats.Add("Bitfinex", p, assets.AssetTypeSpot, 1000, 10000)
	stats.Add("Bitstamp", p, assets.AssetTypeSpot, 1337, 10000)
	exchangeName, err := GetExchangeHighestPriceByCurrencyPair(p, asset)
	if err != nil {
		t.Error(err)
	}

	if exchangeName != "Bitstamp" {
		t.Error("Unexpected result")
	}

	_, err = GetExchangeHighestPriceByCurrencyPair(currency.NewPairFromStrings("BTC", "AUD"), asset)
	if err == nil {
		t.Error("Unexpected result")
	}
}

func TestGetExchangeLowestPriceByCurrencyPair(t *testing.T) {
	SetupTestHelpers(t)

	p := currency.NewPairFromStrings("BTC", "USD")
	asset := assets.AssetTypeSpot
	stats.Add("Bitfinex", p, assets.AssetTypeSpot, 1000, 10000)
	stats.Add("Bitstamp", p, assets.AssetTypeSpot, 1337, 10000)
	exchangeName, err := GetExchangeLowestPriceByCurrencyPair(p, asset)
	if err != nil {
		t.Error(err)
	}

	if exchangeName != "Bitfinex" {
		t.Error("Unexpected result")
	}

	_, err = GetExchangeLowestPriceByCurrencyPair(currency.NewPairFromStrings("BTC", "AUD"), asset)
	if err == nil {
		t.Error("Unexpected reuslt")
	}
}

func TestGetCryptocurrenciesByExchange(t *testing.T) {
	SetupTestHelpers(t)

	_, err := GetCryptocurrenciesByExchange("Bitfinex", false, false, assets.AssetTypeSpot)
	if err != nil {
		t.Fatalf("Test failed. Err %s", err)
	}
}
