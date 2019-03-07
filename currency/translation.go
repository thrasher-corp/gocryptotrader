package currency

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

// GetTranslation returns similar strings for a particular currency if not found
// returns the code back
func GetTranslation(currency Code) (Code, bool) {
	val, ok := translations[currency]
	if !ok {
		return currency, ok
	}
	return val, ok
}
