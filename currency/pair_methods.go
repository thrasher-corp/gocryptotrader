package currency

import (
	"encoding/json"
)

// EMPTYFORMAT defines an empty pair format
var EMPTYFORMAT = PairFormat{}

// String returns a currency pair string
func (p Pair) String() string {
	return p.Base.String() + p.Delimiter + p.Quote.String()
}

// Lower converts the pair object to lowercase
func (p Pair) Lower() Pair {
	p.Base = p.Base.Lower()
	p.Quote = p.Quote.Lower()
	return p
}

// Upper converts the pair object to uppercase
func (p Pair) Upper() Pair {
	p.Base = p.Base.Upper()
	p.Quote = p.Quote.Upper()
	return p
}

// UnmarshalJSON comforms type to the umarshaler interface
func (p *Pair) UnmarshalJSON(d []byte) error {
	var pair string
	err := json.Unmarshal(d, &pair)
	if err != nil {
		return err
	}

	if pair == "" {
		*p = EMPTYPAIR
		return nil
	}

	newPair, err := NewPairFromString(pair)
	if err != nil {
		return err
	}

	*p = newPair
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (p Pair) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// Format changes the currency based on user preferences overriding the default
// String() display
func (p Pair) Format(pf PairFormat) Pair {
	p.Delimiter = pf.Delimiter
	if pf.Uppercase {
		return p.Upper()
	}
	return p.Lower()
}

// Equal compares two currency pairs and returns whether or not they are equal
func (p Pair) Equal(cPair Pair) bool {
	return p.Base.Equal(cPair.Base) && p.Quote.Equal(cPair.Quote)
}

// EqualIncludeReciprocal compares two currency pairs and returns whether or not
// they are the same including reciprocal currencies.
func (p Pair) EqualIncludeReciprocal(cPair Pair) bool {
	return (p.Base.Equal(cPair.Base) && p.Quote.Equal(cPair.Quote)) ||
		(p.Base.Equal(cPair.Quote) && p.Quote.Equal(cPair.Base))
}

// IsCryptoPair checks to see if the pair is a crypto pair e.g. BTCLTC
func (p Pair) IsCryptoPair() bool {
	return p.Base.IsCryptocurrency() && p.Quote.IsCryptocurrency()
}

// IsCryptoFiatPair checks to see if the pair is a crypto fiat pair e.g. BTCUSD
func (p Pair) IsCryptoFiatPair() bool {
	return (p.Base.IsCryptocurrency() && p.Quote.IsFiatCurrency()) ||
		(p.Base.IsFiatCurrency() && p.Quote.IsCryptocurrency())
}

// IsFiatPair checks to see if the pair is a fiat pair e.g. EURUSD
func (p Pair) IsFiatPair() bool {
	return p.Base.IsFiatCurrency() && p.Quote.IsFiatCurrency()
}

// IsCryptoStablePair checks to see if the pair is a crypto stable pair e.g.
// LTC-USDT
func (p Pair) IsCryptoStablePair() bool {
	return (p.Base.IsCryptocurrency() && p.Quote.IsStableCurrency()) ||
		(p.Base.IsStableCurrency() && p.Quote.IsCryptocurrency())
}

// IsStablePair checks to see if the pair is a stable pair e.g. USDT-DAI
func (p Pair) IsStablePair() bool {
	return p.Base.IsStableCurrency() && p.Quote.IsStableCurrency()
}

// IsInvalid checks invalid pair if base and quote are the same
func (p Pair) IsInvalid() bool {
	return p.Base.Equal(p.Quote)
}

// Swap turns the currency pair into its reciprocal
func (p Pair) Swap() Pair {
	return Pair{Base: p.Quote, Quote: p.Base}
}

// IsEmpty returns whether or not the pair is empty or is missing a currency
// code
func (p Pair) IsEmpty() bool {
	return p.Base.IsEmpty() && p.Quote.IsEmpty()
}

// Contains checks to see if a pair contains a specific currency
func (p Pair) Contains(c Code) bool {
	return p.Base.Equal(c) || p.Quote.Equal(c)
}

// Len derives full length for match exclusion.
func (p Pair) Len() int {
	return len(p.Base.String()) + len(p.Quote.String())
}

// Other returns the other currency from pair, if not matched returns empty code.
func (p Pair) Other(c Code) (Code, error) {
	if p.Base.Equal(c) {
		return p.Quote, nil
	}
	if p.Quote.Equal(c) {
		return p.Base, nil
	}
	return EMPTYCODE, ErrCurrencyCodeEmpty
}

// IsPopulated returns true if the currency pair have both non-empty values for base and quote.
func (p Pair) IsPopulated() bool {
	return !p.Base.IsEmpty() && !p.Quote.IsEmpty()
}
