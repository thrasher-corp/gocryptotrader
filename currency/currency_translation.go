package currency

import "errors"

var translations = map[Code]Code{
	BTC:  XBT,
	ETH:  XETH,
	DOGE: XDG,
	USD:  USDT,
	XBT:  BTC,
	XETH: ETH,
	XDG:  DOGE,
	USDT: USD,
}

// GetTranslation returns similar strings for a particular currency
func GetTranslation(currency Code) (Code, error) {
	val, ok := translations[currency]
	if !ok {
		return Code{}, errors.New("no translation found for specified currency")
	}
	return val, nil
}

// HasTranslation returns whether or not a particular currency has a translation
func HasTranslation(currency Code) bool {
	_, ok := translations[currency]
	return ok
}
