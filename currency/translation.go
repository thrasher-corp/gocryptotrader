package currency

// GetTranslation returns similar strings for a particular currency if not found
// returns the code back
func GetTranslation(currency Code) Code {
	val, ok := translations[currency.Item]
	if !ok {
		return currency
	}
	return val
}

var translations = map[*Item]Code{
	BTC.Item:  XBT,
	ETH.Item:  XETH,
	DOGE.Item: XDG,
	USD.Item:  USDT,
	XBT.Item:  BTC,
	XETH.Item: ETH,
	XDG.Item:  DOGE,
	USDT.Item: USD,
}
