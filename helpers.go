package main

import (
	"errors"
	"fmt"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/translation"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

// MapCurrenciesByExchange returns a list of currency pairs mapped to an
// exchange
func MapCurrenciesByExchange(p []pair.CurrencyPair) map[string][]pair.CurrencyPair {
	currencyExchange := make(map[string][]pair.CurrencyPair)
	for x := range p {
		for y := range bot.config.Exchanges {
			exchName := bot.config.Exchanges[y].Name
			success, err := bot.config.SupportsPair(exchName, p[x])
			if err != nil || !success {
				continue
			}

			result, ok := currencyExchange[exchName]
			if !ok {
				var pairs []pair.CurrencyPair
				pairs = append(pairs, p[x])
				currencyExchange[exchName] = pairs
			} else {
				result = append(result, p[x])
				currencyExchange[exchName] = result
			}
		}
	}
	return currencyExchange
}

// GetExchangeNamesByCurrency returns a list of exchanges supporting
// a currency pair based on whether the exchange is enabled or not
func GetExchangeNamesByCurrency(p pair.CurrencyPair, enabled bool) []string {
	var exchanges []string
	for x := range bot.config.Exchanges {
		if enabled != bot.config.Exchanges[x].Enabled {
			continue
		}

		exchName := bot.config.Exchanges[x].Name
		success, err := bot.config.SupportsPair(exchName, p)
		if err != nil {
			continue
		}

		if success {
			exchanges = append(exchanges, exchName)
		}
	}
	return exchanges
}

// GetRelatableCryptocurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g ETHBTC -> ETHLTC -> ETHUSDT -> ETHREP)
// incOrig includes the supplied pair if desired
func GetRelatableCryptocurrencies(p pair.CurrencyPair) []pair.CurrencyPair {
	var pairs []pair.CurrencyPair
	cryptocurrencies := currency.CryptoCurrencies

	for x := range cryptocurrencies {
		newPair := pair.NewCurrencyPair(p.FirstCurrency.String(), cryptocurrencies[x])
		if pair.Contains(pairs, newPair) {
			continue
		}
		pairs = append(pairs, newPair)
	}
	return pairs
}

// GetRelatableFiatCurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g ETHUSD -> ETHAUD -> ETHGBP -> ETHJPY)
// incOrig includes the supplied pair if desired
func GetRelatableFiatCurrencies(p pair.CurrencyPair) []pair.CurrencyPair {
	var pairs []pair.CurrencyPair
	fiatCurrencies := currency.BaseCurrencies

	for x := range fiatCurrencies {
		newPair := pair.NewCurrencyPair(p.FirstCurrency.String(), fiatCurrencies[x])
		if pair.Contains(pairs, newPair) {
			continue
		}
		pairs = append(pairs, newPair)
	}
	return pairs
}

// GetRelatableCurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g BTCUSD -> BTC USDT -> XBT USDT -> XBT USD)
// incOrig includes the supplied pair if desired
func GetRelatableCurrencies(p pair.CurrencyPair, incOrig bool) []pair.CurrencyPair {
	var pairs []pair.CurrencyPair
	if incOrig {
		pairs = append(pairs, p)
	}

	first, err := translation.GetTranslation(p.FirstCurrency)
	if err == nil {
		pairs = append(pairs, pair.NewCurrencyPair(first.String(),
			p.SecondCurrency.String()))

		second, err := translation.GetTranslation(p.SecondCurrency)
		if err == nil {
			pairs = append(pairs, pair.NewCurrencyPair(first.String(),
				second.String()))
		}
	}

	second, err := translation.GetTranslation(p.SecondCurrency)
	if err == nil {
		pairs = append(pairs, pair.NewCurrencyPair(p.FirstCurrency.String(),
			second.String()))
	}
	return pairs
}

// GetSpecificOrderbook returns a specific orderbook given the currency,
// exchangeName and assetType
func GetSpecificOrderbook(currency, exchangeName, assetType string) (orderbook.Base, error) {
	var specificOrderbook orderbook.Base
	var err error
	for x := range bot.exchanges {
		if bot.exchanges[x] != nil {
			if bot.exchanges[x].GetName() == exchangeName {
				specificOrderbook, err = bot.exchanges[x].GetOrderbookEx(
					pair.NewCurrencyPairFromString(currency),
					assetType,
				)
				break
			}
		}
	}
	return specificOrderbook, err
}

// GetSpecificTicker returns a specific ticker given the currency,
// exchangeName and assetType
func GetSpecificTicker(currency, exchangeName, assetType string) (ticker.Price, error) {
	var specificTicker ticker.Price
	var err error
	for x := range bot.exchanges {
		if bot.exchanges[x] != nil {
			if bot.exchanges[x].GetName() == exchangeName {
				specificTicker, err = bot.exchanges[x].GetTickerPrice(
					pair.NewCurrencyPairFromString(currency),
					assetType,
				)
				break
			}
		}
	}
	return specificTicker, err
}

// GetCollatedExchangeAccountInfoByCoin collates individual exchange account
// information and turns into into a map string of
// exchange.AccountCurrencyInfo
func GetCollatedExchangeAccountInfoByCoin(accounts []exchange.AccountInfo) map[string]exchange.AccountCurrencyInfo {
	result := make(map[string]exchange.AccountCurrencyInfo)
	for i := 0; i < len(accounts); i++ {
		for j := 0; j < len(accounts[i].Currencies); j++ {
			currencyName := accounts[i].Currencies[j].CurrencyName
			avail := accounts[i].Currencies[j].TotalValue
			onHold := accounts[i].Currencies[j].Hold

			info, ok := result[currencyName]
			if !ok {
				accountInfo := exchange.AccountCurrencyInfo{CurrencyName: currencyName, Hold: onHold, TotalValue: avail}
				result[currencyName] = accountInfo
			} else {
				info.Hold += onHold
				info.TotalValue += avail
				result[currencyName] = info
			}
		}
	}
	return result
}

// GetAccountCurrencyInfoByExchangeName returns info for an exchange
func GetAccountCurrencyInfoByExchangeName(accounts []exchange.AccountInfo, exchangeName string) (exchange.AccountInfo, error) {
	for i := 0; i < len(accounts); i++ {
		if accounts[i].ExchangeName == exchangeName {
			return accounts[i], nil
		}
	}
	return exchange.AccountInfo{}, errors.New(exchange.ErrExchangeNotFound)
}

// GetExchangeHighestPriceByCurrencyPair returns the exchange with the highest
// price for a given currency pair and asset type
func GetExchangeHighestPriceByCurrencyPair(p pair.CurrencyPair, assetType string) (string, error) {
	result := stats.SortExchangesByPrice(p, assetType, true)
	if len(result) != 1 {
		return "", fmt.Errorf("no stats for supplied currency pair and asset type")
	}

	return result[0].Exchange, nil
}

// GetExchangeLowestPriceByCurrencyPair returns the exchange with the lowest
// price for a given currency pair and asset type
func GetExchangeLowestPriceByCurrencyPair(p pair.CurrencyPair, assetType string) (string, error) {
	result := stats.SortExchangesByPrice(p, assetType, false)
	if len(result) != 1 {
		return "", fmt.Errorf("no stats for supplied currency pair and asset type")
	}

	return result[0].Exchange, nil
}
