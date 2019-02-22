package currency

import (
	"github.com/thrasher-/gocryptotrader/common"
)

// Pair holds currency pair information
type Pair struct {
	Delimiter string `json:"delimiter"`
	Base      Code   `json:"base"`
	Quote     Code   `json:"quote"`
}

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
	err := common.JSONDecode(d, &pair)
	if err != nil {
		return err
	}

	*p = NewPairFromString(pair)
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (p Pair) MarshalJSON() ([]byte, error) {
	return common.JSONEncode(p.String())
}

// Format changes the currency based on user preferences overriding the default
// String() display
func (p Pair) Format(delimiter string, uppercase bool) Pair {
	p.Delimiter = delimiter

	if uppercase {
		return p.Upper()
	}
	return p.Lower()
}

// Equal compares two currency pairs and returns whether or not they are equal
func (p Pair) Equal(cPair Pair) bool {
	if p.Base.Upper() == cPair.Base.Upper() &&
		p.Quote.Upper() == cPair.Quote.Upper() {
		return true
	}
	return false
}

// EqualIncludeReciprocal compares two currency pairs and returns whether or not
// they are the same including reciprocal currencies.
func (p Pair) EqualIncludeReciprocal(cPair Pair) bool {
	if p.Base.Upper() == cPair.Base.Upper() &&
		p.Quote.Upper() == cPair.Quote.Upper() ||
		p.Base.Upper() == cPair.Quote.Upper() &&
			p.Quote.Upper() == cPair.Base.Upper() {
		return true
	}
	return false
}

// IsCrypto checks to see if the pair is a crypto pair e.g. BTCLTC
func (p Pair) IsCrypto() bool {
	return system.IsCryptocurrency(p.Base) &&
		system.IsCryptocurrency(p.Quote)
}

// IsCryptoFiat checks to see if the pair is a crypto fiat pair e.g. BTCUSD
func (p Pair) IsCryptoFiat() bool {
	return system.IsCryptocurrency(p.Base) &&
		system.IsFiatCurrency(p.Quote) ||
		system.IsFiatCurrency(p.Base) &&
			system.IsCryptocurrency(p.Quote)
}

// IsFiat checks to see if the pair is a fiat pair e.g. EURUSD
func (p Pair) IsFiat() bool {
	return system.IsFiatCurrency(p.Base) && system.IsFiatCurrency(p.Quote)
}

// IsInvalid checks invalid pair if base and quote are the same
func (p Pair) IsInvalid() bool {
	return p.Base.C.name == p.Quote.C.name
}

// Swap turns the currency pair into its reciprocal
func (p Pair) Swap() Pair {
	b := p.Base
	p.Base = p.Quote
	p.Quote = b
	return p
}

// IsEmpty returns whether or not the pair is empty or is missing a currency
// code
func (p Pair) IsEmpty() bool {
	return p.Base.IsEmpty() || p.Quote.IsEmpty()
}

// ContainsCurrency checks to see if a pair contains a specific currency
func (p Pair) ContainsCurrency(c Code) bool {
	return p.Base.Upper().String() == c.Upper().String() ||
		p.Quote.Upper().String() == c.Upper().String()
}
