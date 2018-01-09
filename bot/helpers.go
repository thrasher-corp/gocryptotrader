package bot

import (
	"errors"
	"fmt"

	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

//
// GetSpecificOrderbook returns a specific orderbook given the currency,
// exchangeName and assetType
func (b *Bot) GetSpecificOrderbook(currency, exchangeName, assetType string) (orderbook.Base, error) {
	var specificOrderbook orderbook.Base
	var err error
	for i := 0; i < len(b.Exchanges); i++ {
		if b.Exchanges[i] != nil {
			if b.Exchanges[i].IsEnabled() && b.Exchanges[i].GetName() == exchangeName {
				specificOrderbook, err = b.Exchanges[i].GetOrderbookEx(
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
func (b *Bot) GetSpecificTicker(currency, exchangeName, assetType string) (ticker.Price, error) {
	var specificTicker ticker.Price
	var err error
	for i := 0; i < len(b.Exchanges); i++ {
		if b.Exchanges[i] != nil {
			if b.Exchanges[i].IsEnabled() && b.Exchanges[i].GetName() == exchangeName {
				specificTicker, err = b.Exchanges[i].GetTickerPrice(
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
