package currency

import "errors"

var translations = map[Code]Code{
	BTC:  XBT,
	ETH:  XETH,
	DOGE: XDG,
	USD:  USDT,
}

// GetTranslation returns similar strings for a particular currency
func GetTranslation(currency Code) (Code, error) {
	for k, v := range translations {
		if k == currency {
			return v, nil
		}

		if v == currency {
			return k, nil
		}
	}
	return Code{}, errors.New("no translation found for specified currency")
}

// HasTranslation returns whether or not a particular currency has a translation
func HasTranslation(currency Code) bool {
	_, err := GetTranslation(currency)
	return (err == nil)
}
