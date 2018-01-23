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
	for k, v := range translations {
		if k == currency {
			return v, nil
		}

		if v == currency {
			return k, nil
		}
	}
	return "", errors.New("no translation found for specified currency")
}

// HasTranslation returns whether or not a particular currency has a translation
func HasTranslation(currency pair.CurrencyItem) bool {
	_, err := GetTranslation(currency)
	if err != nil {
		return false
	}
	return true
}
