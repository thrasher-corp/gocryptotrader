package translation

import (
	"errors"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

var translations = map[pair.CurrencyItem]pair.CurrencyItem{
	"BTC":  "XBT",
	"ETH":  "XETH",
	"DOGE": "XDG",
	"USD":  "USDT",
}

// GetTranslation returns similar strings for a particular currency
func GetTranslation(currency pair.CurrencyItem) (pair.CurrencyItem, error) {
	result, ok := translations[currency]
	if !ok {
		return "", errors.New("no translation found for specified currency")
	}

	return result, nil
}

// HasTranslation returns whether or not a particular currency has a translation
func HasTranslation(currency pair.CurrencyItem) bool {
	_, ok := translations[currency]
	if !ok {
		return false
	}
	return true
}
