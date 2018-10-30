package engine

import (
	"errors"
	"fmt"
	"log"

	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/translation"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-/gocryptotrader/portfolio"
)

// GetAllAvailablePairs returns a list of all available pairs on either enabled
// or disabled exchanges
func GetAllAvailablePairs(enabledExchangesOnly bool) []pair.CurrencyPair {
	var pairList []pair.CurrencyPair
	for x := range Bot.Config.Exchanges {
		if enabledExchangesOnly && !Bot.Config.Exchanges[x].Enabled {
			continue
		}

		exchName := Bot.Config.Exchanges[x].Name
		pairs, err := Bot.Config.GetAvailablePairs(exchName)
		if err != nil {
			continue
		}

		for y := range pairs {
			if pair.Contains(pairList, pairs[y], false) {
				continue
			}
			pairList = append(pairList, pairs[y])
		}
	}
	return pairList
}

// GetSpecificAvailablePairs returns a list of supported pairs based on specific
// parameters
func GetSpecificAvailablePairs(enabledExchangesOnly, fiatPairs, includeUSDT, cryptoPairs bool) []pair.CurrencyPair {
	var pairList []pair.CurrencyPair
	supportedPairs := GetAllAvailablePairs(enabledExchangesOnly)

	for x := range supportedPairs {
		if fiatPairs {
			if currency.IsCryptoFiatPair(supportedPairs[x]) &&
				!pair.ContainsCurrency(supportedPairs[x], "USDT") ||
				(includeUSDT && pair.ContainsCurrency(supportedPairs[x], "USDT") && currency.IsCryptoPair(supportedPairs[x])) {
				if pair.Contains(pairList, supportedPairs[x], false) {
					continue
				}
				pairList = append(pairList, supportedPairs[x])
			}
		}
		if cryptoPairs {
			if currency.IsCryptoPair(supportedPairs[x]) {
				if pair.Contains(pairList, supportedPairs[x], false) {
					continue
				}
				pairList = append(pairList, supportedPairs[x])
			}
		}
	}
	return pairList
}

// IsRelatablePairs checks to see if the two pairs are relatable
func IsRelatablePairs(p1, p2 pair.CurrencyPair, includeUSDT bool) bool {
	if p1.Equal(p2, false) {
		return true
	}

	var relatablePairs []pair.CurrencyPair
	relatablePairs = GetRelatableCurrencies(p1, true, includeUSDT)

	if currency.IsCryptoFiatPair(p1) {
		for x := range relatablePairs {
			relatablePairs = append(relatablePairs, GetRelatableFiatCurrencies(relatablePairs[x])...)
		}
	}
	return pair.Contains(relatablePairs, p2, false)
}

// MapCurrenciesByExchange returns a list of currency pairs mapped to an
// exchange
func MapCurrenciesByExchange(p []pair.CurrencyPair, enabledExchangesOnly bool) map[string][]pair.CurrencyPair {
	currencyExchange := make(map[string][]pair.CurrencyPair)
	for x := range p {
		for y := range Bot.Config.Exchanges {
			if enabledExchangesOnly && !Bot.Config.Exchanges[y].Enabled {
				continue
			}
			exchName := Bot.Config.Exchanges[y].Name
			success, err := Bot.Config.SupportsPair(exchName, p[x])
			if err != nil || !success {
				continue
			}

			result, ok := currencyExchange[exchName]
			if !ok {
				var pairs []pair.CurrencyPair
				pairs = append(pairs, p[x])
				currencyExchange[exchName] = pairs
			} else {
				if pair.Contains(result, p[x], false) {
					continue
				}
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
	for x := range Bot.Config.Exchanges {
		if enabled != Bot.Config.Exchanges[x].Enabled {
			continue
		}

		exchName := Bot.Config.Exchanges[x].Name
		success, err := Bot.Config.SupportsPair(exchName, p)
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
		if pair.Contains(pairs, newPair, false) {
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
	fiatCurrencies := currency.FiatCurrencies

	for x := range fiatCurrencies {
		newPair := pair.NewCurrencyPair(p.FirstCurrency.String(), fiatCurrencies[x])
		if pair.Contains(pairs, newPair, false) {
			continue
		}
		pairs = append(pairs, newPair)
	}
	return pairs
}

// GetRelatableCurrencies returns a list of currency pairs if it can find
// any relatable currencies (e.g BTCUSD -> BTC USDT -> XBT USDT -> XBT USD)
// incOrig includes the supplied pair if desired
func GetRelatableCurrencies(p pair.CurrencyPair, incOrig, incUSDT bool) []pair.CurrencyPair {
	var pairs []pair.CurrencyPair

	addPair := func(p pair.CurrencyPair) {
		if pair.Contains(pairs, p, true) {
			return
		}
		pairs = append(pairs, p)
	}

	buildPairs := func(p pair.CurrencyPair, incOrig bool) {
		if incOrig {
			addPair(p)
		}

		first, err := translation.GetTranslation(p.FirstCurrency)
		if err == nil {
			addPair(pair.NewCurrencyPair(first.String(),
				p.SecondCurrency.String()))

			second, err := translation.GetTranslation(p.SecondCurrency)
			if err == nil {
				addPair(pair.NewCurrencyPair(first.String(),
					second.String()))
			}
		}

		second, err := translation.GetTranslation(p.SecondCurrency)
		if err == nil {
			addPair(pair.NewCurrencyPair(p.FirstCurrency.String(),
				second.String()))
		}
	}

	buildPairs(p, incOrig)
	buildPairs(p.Swap(), incOrig)

	if !incUSDT {
		pairs = pair.RemovePairsByFilter(pairs, "USDT")
	}

	return pairs
}

// GetSpecificOrderbook returns a specific orderbook given the currency,
// exchangeName and assetType
func GetSpecificOrderbook(currency, exchangeName, assetType string) (orderbook.Base, error) {
	var specificOrderbook orderbook.Base
	var err error
	for x := range Bot.Exchanges {
		if Bot.Exchanges[x] != nil {
			if Bot.Exchanges[x].GetName() == exchangeName {
				specificOrderbook, err = Bot.Exchanges[x].FetchOrderbook(
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
	for x := range Bot.Exchanges {
		if Bot.Exchanges[x] != nil {
			if Bot.Exchanges[x].GetName() == exchangeName {
				specificTicker, err = Bot.Exchanges[x].FetchTicker(
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
	if len(result) == 0 {
		return "", fmt.Errorf("no stats for supplied currency pair and asset type")
	}

	return result[0].Exchange, nil
}

// GetExchangeLowestPriceByCurrencyPair returns the exchange with the lowest
// price for a given currency pair and asset type
func GetExchangeLowestPriceByCurrencyPair(p pair.CurrencyPair, assetType string) (string, error) {
	result := stats.SortExchangesByPrice(p, assetType, false)
	if len(result) == 0 {
		return "", fmt.Errorf("no stats for supplied currency pair and asset type")
	}

	return result[0].Exchange, nil
}

// SeedExchangeAccountInfo seeds account info
func SeedExchangeAccountInfo(data []exchange.AccountInfo) {
	if len(data) == 0 {
		return
	}

	port := portfolio.GetPortfolio()

	for i := 0; i < len(data); i++ {
		exchangeName := data[i].ExchangeName
		for j := 0; j < len(data[i].Currencies); j++ {
			currencyName := data[i].Currencies[j].CurrencyName
			onHold := data[i].Currencies[j].Hold
			avail := data[i].Currencies[j].TotalValue
			total := onHold + avail

			if !port.ExchangeAddressExists(exchangeName, currencyName) {
				if total <= 0 {
					continue
				}
				log.Printf("Portfolio: Adding new exchange address: %s, %s, %f, %s\n",
					exchangeName, currencyName, total, portfolio.PortfolioAddressExchange)
				port.Addresses = append(
					port.Addresses,
					portfolio.Address{Address: exchangeName, CoinType: currencyName,
						Balance: total, Description: portfolio.PortfolioAddressExchange},
				)
			} else {
				if total <= 0 {
					log.Printf("Portfolio: Removing %s %s entry.\n", exchangeName,
						currencyName)
					port.RemoveExchangeAddress(exchangeName, currencyName)
				} else {
					balance, ok := port.GetAddressBalance(exchangeName, currencyName, portfolio.PortfolioAddressExchange)
					if !ok {
						continue
					}
					if balance != total {
						log.Printf("Portfolio: Updating %s %s entry with balance %f.\n",
							exchangeName, currencyName, total)
						port.UpdateExchangeAddressBalance(exchangeName, currencyName, total)
					}
				}
			}
		}
	}
}
