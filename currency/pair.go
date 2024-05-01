package currency

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var errCannotCreatePair = errors.New("cannot create currency pair")

// NewBTCUSDT is a shortcut for NewPair(BTC, USDT)
func NewBTCUSDT() Pair {
	return NewPair(BTC, USDT)
}

// NewBTCUSD is a shortcut for NewPair(BTC, USD)
func NewBTCUSD() Pair {
	return NewPair(BTC, USD)
}

// NewPairDelimiter splits the desired currency string at delimiter, the returns
// a Pair struct
func NewPairDelimiter(currencyPair, delimiter string) (Pair, error) {
	if !strings.Contains(currencyPair, delimiter) {
		return EMPTYPAIR,
			fmt.Errorf("delimiter: [%s] not found in currencypair string", delimiter)
	}
	result := strings.Split(currencyPair, delimiter)
	if len(result) < 2 {
		return EMPTYPAIR,
			fmt.Errorf("supplied pair: [%s] cannot be split with %s",
				currencyPair,
				delimiter)
	}
	if len(result) > 2 {
		result[1] = strings.Join(result[1:], delimiter)
	}
	return Pair{
		Delimiter: delimiter,
		Base:      NewCode(result[0]),
		Quote:     NewCode(result[1]),
	}, nil
}

// NewPairFromStrings returns a CurrencyPair without a delimiter
func NewPairFromStrings(base, quote string) (Pair, error) {
	if strings.Contains(base, " ") {
		return EMPTYPAIR,
			fmt.Errorf("cannot create pair, invalid base currency string [%s]",
				base)
	}

	if strings.Contains(quote, " ") {
		return EMPTYPAIR,
			fmt.Errorf("cannot create pair, invalid quote currency string [%s]",
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

// NewPairFromString converts currency string into a new CurrencyPair
// with or without delimiter
func NewPairFromString(currencyPair string) (Pair, error) {
	if len(currencyPair) < 3 {
		return EMPTYPAIR,
			fmt.Errorf("%w from %s string too short to be a currency pair",
				errCannotCreatePair,
				currencyPair)
	}

	for x := range currencyPair {
		if unicode.IsPunct(rune(currencyPair[x])) {
			return Pair{
				Base:      NewCode(currencyPair[:x]),
				Delimiter: string(currencyPair[x]),
				Quote:     NewCode(currencyPair[x+1:]),
			}, nil
		}
	}

	return NewPairFromStrings(currencyPair[0:3], currencyPair[3:])
}

// NewPairFromFormattedPairs matches a supplied currency pair to a list of pairs
// with a specific format. This is helpful for exchanges which
// provide currency pairs with no delimiter so we can match it with a list and
// apply the same format
func NewPairFromFormattedPairs(currencyPair string, pairs Pairs, pairFmt PairFormat) (Pair, error) {
	for x := range pairs {
		if strings.EqualFold(pairFmt.Format(pairs[x]), currencyPair) {
			return pairs[x], nil
		}
	}
	return NewPairFromString(currencyPair)
}

// Format formats the given pair as a string
func (f PairFormat) Format(pair Pair) string {
	return pair.Format(f).String()
}

// MatchPairsWithNoDelimiter will move along a predictable index on the provided currencyPair
// it will then split on that index and verify whether that currencypair exists in the
// supplied pairs
// this allows for us to match strange currencies with no delimiter where it is difficult to
// infer where the delimiter is located eg BETHERETH is BETHER ETH
func MatchPairsWithNoDelimiter(currencyPair string, pairs Pairs, pairFmt PairFormat) (Pair, error) {
	for i := range pairs {
		fPair := pairs[i].Format(pairFmt)
		maxLen := 6
		if len(currencyPair) < maxLen {
			maxLen = len(currencyPair)
		}
		for j := 1; j <= maxLen; j++ {
			if fPair.Base.String() == currencyPair[0:j] &&
				fPair.Quote.String() == currencyPair[j:] {
				return fPair, nil
			}
		}
	}
	return EMPTYPAIR, fmt.Errorf("currency %v not found in supplied pairs", currencyPair)
}

// GetFormatting returns the formatting style of a pair
func (p Pair) GetFormatting() (PairFormat, error) {
	if p.Base.UpperCase != p.Quote.UpperCase {
		return PairFormat{}, fmt.Errorf("%w casing mismatch", errPairFormattingInconsistent)
	}
	return PairFormat{
		Uppercase: p.Base.UpperCase,
		Delimiter: p.Delimiter,
	}, nil
}
