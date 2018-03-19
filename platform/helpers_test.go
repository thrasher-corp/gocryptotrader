package platform

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
)

var (
	pairs []pair.CurrencyPair
	p1    pair.CurrencyPair
)

func TestSetup(t *testing.T) {
	newbot.SetCurrencyProvider()
	newbot.RetrieveCurrencyPairs()
	p1 = pair.NewCurrencyPair("btc", "usd")
	p2 := pair.NewCurrencyPair("ltc", "usd")
	pairs = append(pairs, p1, p2)
}

func TestMapCurrenciesByExchange(t *testing.T) {
	newPairMap := newbot.MapCurrenciesByExchange(pairs)
	if len(newPairMap) != 16 {
		t.Error("test failed - helpers MapCurrenciesByExchange() error")
	}
}

func TestGetExchangeNamesByCurrency(t *testing.T) {
	supportedExchs := newbot.GetExchangeNamesByCurrency(p1, true)
	if len(supportedExchs) != 14 {
		t.Error("test failed - helpers GetExchangeNamesByCurrency() error")
	}
	NonEnabledExchs := newbot.GetExchangeNamesByCurrency(p1, false)
	if len(NonEnabledExchs) != 1 {
		t.Error("test failed - helpers GetExchangeNamesByCurrency() error")
	}
}

func TestGetRelatableCryptocurrencies(t *testing.T) {
	relatablePair := newbot.GetRelatableCryptocurrencies(p1)
	if len(relatablePair) != 522 {
		t.Error("test failed - helpers GetRelatableCryptocurrencies() error")
	}
}

func TestGetRelatableFiatCurrencies(t *testing.T) {
	relatablePair := newbot.GetRelatableFiatCurrencies(p1)
	if len(relatablePair) != 28 {
		t.Error("test failed - helpers GetRelatableFiatCurrencies() error")
	}
}

func TestGetRelatableCurrencies(t *testing.T) {
	relatablePair := newbot.GetRelatableCurrencies(p1, false)
	if len(relatablePair) != 0 {
		t.Error("test failed - helpers GetRelatableCurrencies() error")
	}
	relatablePair = newbot.GetRelatableCurrencies(p1, true)
	if len(relatablePair) != 1 {
		t.Error("test failed - helpers GetRelatableCurrencies() error")
	}
}

func TestGetSpecificOrderbook(t *testing.T) {
	_, err := newbot.GetSpecificOrderbook("bla", "ning-nong", "something")
	if err != nil {
		t.Error("test failed - helpers GetSpecificOrderbook() error", err)
	}
}

func TestGetSpecificTicker(t *testing.T) {
	_, err := newbot.GetSpecificTicker("bla", "ning-nong", "something")
	if err != nil {
		t.Error("test failed - helpers GetSpecificTicker() error", err)
	}
}

func TestGetCollatedExchangeAccountInfoByCoin(t *testing.T) {
	eai := exchange.AccountInfo{
		ExchangeName: "testExchange",
		Currencies: []exchange.AccountCurrencyInfo{
			exchange.AccountCurrencyInfo{CurrencyName: "test", TotalValue: 100, Hold: 100},
		},
	}

	newMap := newbot.GetCollatedExchangeAccountInfoByCoin([]exchange.AccountInfo{eai})
	if _, ok := newMap["test"]; !ok {
		t.Error("test failed - helpers GetCollatedExchangeAccountInfoByCoin() error")
	}
}

func TestGetAccountCurrencyInfoByExchangeName(t *testing.T) {
	eai := exchange.AccountInfo{
		ExchangeName: "testExchange",
		Currencies: []exchange.AccountCurrencyInfo{
			exchange.AccountCurrencyInfo{CurrencyName: "test", TotalValue: 100, Hold: 100},
		},
	}

	ai, err := newbot.GetAccountCurrencyInfoByExchangeName([]exchange.AccountInfo{eai}, "testExchange")
	if err != nil {
		t.Error("test failed - helpers GetAccountCurrencyInfoByExchangeName() error", err)
	}
	if ai.ExchangeName != "testExchange" {
		t.Error("test failed - helpers GetAccountCurrencyInfoByExchangeName() error")
	}

	ai, err = newbot.GetAccountCurrencyInfoByExchangeName([]exchange.AccountInfo{eai}, "bla")
	if err == nil {
		t.Error("test failed - helpers GetAccountCurrencyInfoByExchangeName() error", err)
	}
}

func TestGetExchangeHighestPriceByCurrencyPair(t *testing.T) {
	_, err := newbot.GetExchangeHighestPriceByCurrencyPair(p1, "SPOT")
	if err == nil {
		t.Error("test failed - helpers GetExchangeHighestPriceByCurrencyPair() error")
	}
}

func TestGetExchangeLowestPriceByCurrencyPair(t *testing.T) {
	_, err := newbot.GetExchangeLowestPriceByCurrencyPair(p1, "SPOT")
	if err == nil {
		t.Error("test failed - helpers GetExchangeLowestPriceByCurrencyPair() error")
	}
}
