package currency

import (
	"fmt"
	"strings"
)

// NewPairDelimiter splits the desired currency string at delimeter, the returns
// a Pair struct
func NewPairDelimiter(currencyPair, delimiter string) Pair {
	result := strings.Split(currencyPair, delimiter)
	return Pair{
		Delimiter: delimiter,
		Base:      NewCode(result[0]),
		Quote:     NewCode(result[1]),
	}
}

// NewPairFromStrings returns a CurrencyPair without a delimiter
func NewPairFromStrings(base, quote string) (Pair, error) {
	if strings.Contains(base, " ") {
		return Pair{},
			fmt.Errorf("cannot create pair invalid base currency string [%s]",
				base)
	}

	if strings.Contains(quote, " ") {
		return Pair{},
			fmt.Errorf("cannot create pair invalid quote currency string [%s]",
				quote)
	}

	return Pair{Base: NewCode(base), Quote: NewCode(quote)}, nil
}

// NewPair returns a currency pair from currency codes
func NewPair(baseCurrency, quoteCurrency Code) Pair {
	return Pair{
		Base:  baseCurrency,
		Quote: quoteCurrency,
	}
}

// NewPairWithDelimiter returns a CurrencyPair with a delimiter
func NewPairWithDelimiter(base, quote, delimiter string) Pair {
	return Pair{
		Base:      NewCode(base),
		Quote:     NewCode(quote),
		Delimiter: delimiter,
	}
}

// NewPairFromIndex returns a CurrencyPair via a currency string and specific
// index
func NewPairFromIndex(currencyPair, index string) (Pair, error) {
	i := strings.Index(currencyPair, index)
	if i == -1 {
		return Pair{},
			fmt.Errorf("index %s not found in currency pair string", index)
	}
	if i == 0 {
		return NewPairFromStrings(currencyPair[0:len(index)],
			currencyPair[len(index):])
	}
	return NewPairFromStrings(currencyPair[0:i], currencyPair[i:])
}

// NewPairFromString converts currency string into a new CurrencyPair
// with or without delimeter
func NewPairFromString(currencyPair string) (Pair, error) {
	for x := range delimiters {
		if strings.Contains(currencyPair, delimiters[x]) {
			return NewPairDelimiter(currencyPair, delimiters[x]), nil
		}
	}
	if len(currencyPair) < 3 {
		return Pair{},
			fmt.Errorf("cannot produce a currency pair from %s string",
				currencyPair)
	}
	return NewPairFromStrings(currencyPair[0:3], currencyPair[3:])
}

// NewPairFromFormattedPairs matches a supplied currency pair to a list of pairs
// with a specific format. This is helpful for exchanges which
// provide currency pairs with no delimiter so we can match it with a list and
// apply the same format
func NewPairFromFormattedPairs(currencyPair string, pairs Pairs, pairFmt PairFormat) (Pair, error) {
	for x := range pairs {
		if strings.EqualFold(pairs[x].Format(pairFmt.Delimiter,
			pairFmt.Uppercase).String(), currencyPair) {
			return pairs[x], nil
		}
	}
	return NewPairFromString(currencyPair)
}
