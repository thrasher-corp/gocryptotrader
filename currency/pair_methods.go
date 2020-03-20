package currency

import (
	"encoding/json"
	"strings"
)

// String returns a currency pair string
func (p Pair) String() string {
	return p.Base.String() + p.Delimiter + p.Quote.String()
}

// Lower converts the pair object to lowercase
func (p Pair) Lower() Pair {
	return Pair{
		Delimiter: p.Delimiter,
		Base:      p.Base.Lower(),
		Quote:     p.Quote.Lower(),
	}
}

// Upper converts the pair object to uppercase
func (p Pair) Upper() Pair {
	return Pair{
		Delimiter: p.Delimiter,
		Base:      p.Base.Upper(),
		Quote:     p.Quote.Upper(),
	}
}

// UnmarshalJSON comforms type to the umarshaler interface
func (p *Pair) UnmarshalJSON(d []byte) error {
	var pair string
	err := json.Unmarshal(d, &pair)
	if err != nil {
		return err
	}

	newPair, err := NewPairFromString(pair)
	if err != nil {
		return err
	}

	p.Base = newPair.Base
	p.Quote = newPair.Quote
	p.Delimiter = newPair.Delimiter
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (p Pair) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

// Format changes the currency based on user preferences overriding the default
// String() display
func (p Pair) Format(delimiter string, uppercase bool) Pair {
	newP := Pair{Base: p.Base, Quote: p.Quote, Delimiter: delimiter}
	if uppercase {
		return newP.Upper()
	}
	return newP.Lower()
}

// Equal compares two currency pairs and returns whether or not they are equal
func (p Pair) Equal(cPair Pair) bool {
	return strings.EqualFold(p.Base.String(), cPair.Base.String()) &&
		strings.EqualFold(p.Quote.String(), cPair.Quote.String())
}

// EqualIncludeReciprocal compares two currency pairs and returns whether or not
// they are the same including reciprocal currencies.
func (p Pair) EqualIncludeReciprocal(cPair Pair) bool {
	if (p.Base.Item == cPair.Base.Item && p.Quote.Item == cPair.Quote.Item) ||
		(p.Base.Item == cPair.Quote.Item && p.Quote.Item == cPair.Base.Item) {
		return true
	}
	return false
}

// IsCryptoPair checks to see if the pair is a crypto pair e.g. BTCLTC
func (p Pair) IsCryptoPair() bool {
	return storage.IsCryptocurrency(p.Base) &&
		storage.IsCryptocurrency(p.Quote)
}

// IsCryptoFiatPair checks to see if the pair is a crypto fiat pair e.g. BTCUSD
func (p Pair) IsCryptoFiatPair() bool {
	return (storage.IsCryptocurrency(p.Base) && storage.IsFiatCurrency(p.Quote)) ||
		(storage.IsFiatCurrency(p.Base) && storage.IsCryptocurrency(p.Quote))
}

// IsFiatPair checks to see if the pair is a fiat pair e.g. EURUSD
func (p Pair) IsFiatPair() bool {
	return storage.IsFiatCurrency(p.Base) && storage.IsFiatCurrency(p.Quote)
}

// IsInvalid checks invalid pair if base and quote are the same
func (p Pair) IsInvalid() bool {
	return p.Base.Item == p.Quote.Item
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

// ContainsCurrency checks to see if a pair contains a specific currency
func (p Pair) ContainsCurrency(c Code) bool {
	return p.Base.Item == c.Item || p.Quote.Item == c.Item
}
