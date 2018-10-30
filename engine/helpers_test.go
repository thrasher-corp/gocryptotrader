package engine

import (
	"log"
	"testing"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	TestConfig = "./testdata/configtest.json"
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
			err := Bot.Config.LoadConfig("./testdata/configtest.json")
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
	result := GetSpecificAvailablePairs(true, true, true, false)

	if !pair.Contains(result, pair.NewCurrencyPair("BTC", "USD"), true) {
		t.Fatal("Unexpected result")
	}

	if !pair.Contains(result, pair.NewCurrencyPair("BTC", "USDT"), false) {
		t.Fatal("Unexpected result")
	}

	result = GetSpecificAvailablePairs(true, true, false, false)

	if pair.Contains(result, pair.NewCurrencyPair("BTC", "USDT"), false) {
		t.Fatal("Unexpected result")
	}

	result = GetSpecificAvailablePairs(true, false, false, true)
	if !pair.Contains(result, pair.NewCurrencyPair("LTC", "BTC"), false) {
		t.Fatal("Unexpected result")
	}
}

func TestIsRelatablePairs(t *testing.T) {
	SetupTestHelpers(t)

	// Test relational pairs with similar names
	result := IsRelatablePairs(pair.NewCurrencyPair("XBT", "USD"), pair.NewCurrencyPair("BTC", "USD"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names reversed
	result = IsRelatablePairs(pair.NewCurrencyPair("BTC", "USD"), pair.NewCurrencyPair("XBT", "USD"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names but with Tether support disabled
	result = IsRelatablePairs(pair.NewCurrencyPair("XBT", "USD"), pair.NewCurrencyPair("BTC", "USDT"), false)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with similar names but with Tether support enabled
	result = IsRelatablePairs(pair.NewCurrencyPair("XBT", "USDT"), pair.NewCurrencyPair("BTC", "USD"), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with different ordering, a delimiter and with Tether support enabled
	result = IsRelatablePairs(pair.NewCurrencyPair("AE", "USDT"), pair.NewCurrencyPairDelimiter("USDT-AE", "-"), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relational pairs with different ordering, a delimiter and with Tether support disabled
	result = IsRelatablePairs(pair.NewCurrencyPair("AE", "USDT"), pair.NewCurrencyPairDelimiter("USDT-AE", "-"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names and different fiat currencies
	result = IsRelatablePairs(pair.NewCurrencyPair("XBT", "EUR"), pair.NewCurrencyPair("BTC", "AUD"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names, different fiat currencies and with different ordering
	result = IsRelatablePairs(pair.NewCurrencyPair("USD", "BTC"), pair.NewCurrencyPair("BTC", "EUR"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl pairs with similar names, different fiat currencies and with Tether enabled
	result = IsRelatablePairs(pair.NewCurrencyPair("USD", "BTC"), pair.NewCurrencyPair("BTC", "USDT"), true)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar names
	result = IsRelatablePairs(pair.NewCurrencyPair("LTC", "BTC"), pair.NewCurrencyPair("BTC", "LTC"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar different pairs
	result = IsRelatablePairs(pair.NewCurrencyPair("LTC", "ETH"), pair.NewCurrencyPair("BTC", "ETH"), false)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with similar different pairs and with USDT enabled
	result = IsRelatablePairs(pair.NewCurrencyPair("USDT", "USD"), pair.NewCurrencyPair("BTC", "USD"), true)
	if result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with with similar names
	result = IsRelatablePairs(pair.NewCurrencyPair("XBT", "LTC"), pair.NewCurrencyPair("BTC", "LTC"), false)
	if !result {
		t.Fatal("Unexpected result")
	}

	// Test relationl crypto pairs with different ordering and similar names
	result = IsRelatablePairs(pair.NewCurrencyPair("LTC", "XBT"), pair.NewCurrencyPair("BTC", "LTC"), false)
	if !result {
		t.Fatal("Unexpected result")
	}
}

func TestGetRelatableCryptocurrencies(t *testing.T) {
	SetupTestHelpers(t)
	p := GetRelatableCryptocurrencies(pair.NewCurrencyPair("BTC", "LTC"))
	if !pair.Contains(p, pair.NewCurrencyPair("BTC", "ETH"), true) {
		t.Fatal("Unexpected result")
	}

	backup := currency.CryptoCurrencies
	currency.CryptoCurrencies = append(currency.CryptoCurrencies, "BTC")

	p = GetRelatableCryptocurrencies(pair.NewCurrencyPair("BTC", "LTC"))
	if !pair.Contains(p, pair.NewCurrencyPair("BTC", "ETH"), true) {
		t.Fatal("Unexpected result")
	}

	currency.CryptoCurrencies = backup
}

func TestGetRelatableFiatCurrencies(t *testing.T) {
	SetupTestHelpers(t)
	p := GetRelatableFiatCurrencies(pair.NewCurrencyPair("BTC", "USD"))
	if !pair.Contains(p, pair.NewCurrencyPair("BTC", "EUR"), true) {
		t.Fatal("Unexpected result")
	}

	backup := currency.FiatCurrencies
	currency.FiatCurrencies = append(currency.FiatCurrencies, "USD")

	p = GetRelatableFiatCurrencies(pair.NewCurrencyPair("BTC", "USD"))
	if !pair.Contains(p, pair.NewCurrencyPair("BTC", "ZAR"), true) {
		t.Fatal("Unexpected result")
	}

	currency.FiatCurrencies = backup
}

func TestMapCurrenciesByExchange(t *testing.T) {
	SetupTestHelpers(t)

	var pairs []pair.CurrencyPair
	pairs = append(pairs, pair.NewCurrencyPair("BTC", "USD"))
	pairs = append(pairs, pair.NewCurrencyPair("BTC", "EUR"))

	result := MapCurrenciesByExchange(pairs, true)
	pairs, ok := result["Bitstamp"]
	if !ok {
		t.Fatal("Unexpected result")
	}

	log.Println(pairs)
	if len(pairs) != 2 {
		t.Fatal("Unexpected result")
	}
}

func TestGetExchangeNamesByCurrency(t *testing.T) {
	SetupTestHelpers(t)

	result := GetExchangeNamesByCurrency(pair.NewCurrencyPair("BTC", "USD"), true)
	if !common.StringDataCompare(result, "Bitstamp") {
		t.Fatal("Unexpected result")
	}

	result = GetExchangeNamesByCurrency(pair.NewCurrencyPair("BTC", "JPY"), true)
	if !common.StringDataCompare(result, "Bitflyer") {
		t.Fatal("Unexpected result")
	}

	result = GetExchangeNamesByCurrency(pair.NewCurrencyPair("blah", "JPY"), true)
	if len(result) > 0 {
		t.Fatal("Unexpected result")
	}
}

func TestGetSpecificOrderbook(t *testing.T) {
	SetupTestHelpers(t)

	LoadExchange("Bitstamp", false, nil)
	p := pair.NewCurrencyPair("BTC", "USD")
	bids := []orderbook.Item{}
	bids = append(bids, orderbook.Item{Price: 1000, Amount: 1})

	orderbook.ProcessOrderbook("Bitstamp", p, orderbook.Base{Pair: p, Bids: bids}, ticker.Spot)
	ob, err := GetSpecificOrderbook("BTCUSD", "Bitstamp", ticker.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if ob.Bids[0].Price != 1000 {
		t.Fatal("Unexpected result")
	}

	ob, err = GetSpecificOrderbook("ETHLTC", "Bitstamp", ticker.Spot)
	if err == nil {
		t.Fatal("Unexpected result")
	}

	UnloadExchange("Bitstamp")
}

func TestGetSpecificTicker(t *testing.T) {
	SetupTestHelpers(t)

	LoadExchange("Bitstamp", false, nil)
	p := pair.NewCurrencyPair("BTC", "USD")
	ticker.ProcessTicker("Bitstamp", p, ticker.Price{Last: 1000}, ticker.Spot)

	tick, err := GetSpecificTicker("BTCUSD", "Bitstamp", ticker.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if tick.Last != 1000 {
		t.Fatal("Unexpected result")
	}

	tick, err = GetSpecificTicker("ETHLTC", "Bitstamp", ticker.Spot)
	if err == nil {
		t.Fatal("Unexpected result")
	}

	UnloadExchange("Bitstamp")
}

func TestGetCollatedExchangeAccountInfoByCoin(t *testing.T) {
	SetupTestHelpers(t)

	exchangeInfo := []exchange.AccountInfo{}
	var info exchange.AccountInfo

	info.ExchangeName = "Bitfinex"
	info.Currencies = append(info.Currencies,
		exchange.AccountCurrencyInfo{CurrencyName: "BTC", TotalValue: 100, Hold: 0})
	exchangeInfo = append(exchangeInfo, info)

	info.ExchangeName = "Bitstamp"
	info.Currencies = append(info.Currencies, exchange.AccountCurrencyInfo{CurrencyName: "LTC", TotalValue: 100, Hold: 0})
	exchangeInfo = append(exchangeInfo, info)

	result := GetCollatedExchangeAccountInfoByCoin(exchangeInfo)
	if len(result) == 0 {
		t.Fatal("Unexpected result")
	}

	amount, ok := result["BTC"]
	if !ok {
		t.Fatal("Expected currency was not found in result map")
	}

	if amount.TotalValue != 200 {
		t.Fatal("Unexpected result")
	}

	_, ok = result["ETH"]
	if ok {
		t.Fatal("Unexpected result")
	}
}

func TestGetAccountCurrencyInfoByExchangeName(t *testing.T) {
	SetupTestHelpers(t)

	exchangeInfo := []exchange.AccountInfo{}
	var info exchange.AccountInfo
	info.ExchangeName = "Bitfinex"
	info.Currencies = append(info.Currencies,
		exchange.AccountCurrencyInfo{CurrencyName: "BTC", TotalValue: 100, Hold: 0})
	exchangeInfo = append(exchangeInfo, info)

	result, err := GetAccountCurrencyInfoByExchangeName(exchangeInfo, "Bitfinex")
	if err != nil {
		t.Fatal(err)
	}

	if result.ExchangeName != "Bitfinex" {
		t.Fatal("Unexepcted result")
	}

	_, err = GetAccountCurrencyInfoByExchangeName(exchangeInfo, "ASDF")
	if err.Error() != exchange.ErrExchangeNotFound {
		t.Fatal("Unexepcted result")
	}
}

func TestGetExchangeHighestPriceByCurrencyPair(t *testing.T) {
	SetupTestHelpers(t)

	p := pair.NewCurrencyPair("BTC", "USD")
	stats.Add("Bitfinex", p, ticker.Spot, 1000, 10000)
	stats.Add("Bitstamp", p, ticker.Spot, 1337, 10000)
	exchange, err := GetExchangeHighestPriceByCurrencyPair(p, ticker.Spot)
	if err != nil {
		log.Fatal(err)
	}

	if exchange != "Bitstamp" {
		log.Fatal("Unexpected result")
	}

	_, err = GetExchangeHighestPriceByCurrencyPair(pair.NewCurrencyPair("BTC", "AUD"), ticker.Spot)
	if err == nil {
		log.Fatal("Unexpected reuslt")
	}
}

func TestGetExchangeLowestPriceByCurrencyPair(t *testing.T) {
	SetupTestHelpers(t)

	p := pair.NewCurrencyPair("BTC", "USD")
	stats.Add("Bitfinex", p, ticker.Spot, 1000, 10000)
	stats.Add("Bitstamp", p, ticker.Spot, 1337, 10000)
	exchange, err := GetExchangeLowestPriceByCurrencyPair(p, ticker.Spot)
	if err != nil {
		log.Fatal(err)
	}

	if exchange != "Bitfinex" {
		log.Fatal("Unexpected result")
	}

	_, err = GetExchangeLowestPriceByCurrencyPair(pair.NewCurrencyPair("BTC", "AUD"), ticker.Spot)
	if err == nil {
		log.Fatal("Unexpected reuslt")
	}
}
