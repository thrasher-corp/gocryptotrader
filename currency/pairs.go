package currency

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/log"
)

var (
	errSymbolEmpty = errors.New("symbol is empty")
	errPairsEmpty  = errors.New("pairs are empty")
	errNoDelimiter = errors.New("no delimiter was supplied")
)

// NewPairsFromStrings takes in currency pair strings and returns a currency
// pair list
func NewPairsFromStrings(pairs []string) (Pairs, error) {
	allThePairs := make(Pairs, len(pairs))
	var err error
	for i := range pairs {
		allThePairs[i], err = NewPairFromString(pairs[i])
		if err != nil {
			return nil, err
		}
	}
	return allThePairs, nil
}

// NewPairsFromString takes in a delimiter string and returns a Pairs
// type
func NewPairsFromString(pairs, delimiter string) (Pairs, error) {
	if delimiter == "" {
		return nil, errNoDelimiter
	}
	return NewPairsFromStrings(strings.Split(pairs, delimiter))
}

// Strings returns a slice of strings referring to each currency pair
func (p Pairs) Strings() []string {
	list := make([]string, len(p))
	for i := range p {
		list[i] = p[i].String()
	}
	return list
}

// Join returns a comma separated list of currency pairs
func (p Pairs) Join() string {
	return strings.Join(p.Strings(), ",")
}

// Format formats the pair list to the exchange format configuration
func (p Pairs) Format(delimiter, index string, uppercase bool) Pairs {
	pairs := make(Pairs, 0, len(p))
	var err error
	for _, format := range p {
		if index != "" {
			format, err = NewPairFromIndex(format.String(), index)
			if err != nil {
				log.Errorf(log.Global,
					"failed to create NewPairFromIndex. Err: %s\n", err)
				continue
			}
		}
		format.Delimiter = delimiter
		if uppercase {
			pairs = append(pairs, format.Upper())
		} else {
			pairs = append(pairs, format.Lower())
		}
	}
	return pairs
}

// UnmarshalJSON comforms type to the umarshaler interface
func (p *Pairs) UnmarshalJSON(d []byte) error {
	var pairs string
	err := json.Unmarshal(d, &pairs)
	if err != nil {
		return err
	}

	// If no pairs enabled in config just continue
	if pairs == "" {
		return nil
	}

	*p, err = NewPairsFromString(pairs, ",")
	return err
}

// MarshalJSON conforms type to the marshaler interface
func (p Pairs) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Join())
}

// Upper updates the original pairs and returns the pairs for convenience if
// needed.
func (p Pairs) Upper() Pairs {
	for i := range p {
		p[i] = p[i].Upper()
	}
	return p
}

// Lower updates the original pairs and returns the pairs for convenience if
// needed.
func (p Pairs) Lower() Pairs {
	for i := range p {
		p[i] = p[i].Lower()
	}
	return p
}

// Contains checks to see if a specified pair exists inside a currency pair
// array
func (p Pairs) Contains(check Pair, exact bool) bool {
	for i := range p {
		if exact {
			if p[i].Equal(check) {
				return true
			}
		} else {
			if p[i].EqualIncludeReciprocal(check) {
				return true
			}
		}
	}
	return false
}

// ContainsCurrency checks to see if a specified currency code exists inside a
// currency pair array
func (p Pairs) ContainsCurrency(check Code) bool {
	for i := range p {
		if p[i].Contains(check) {
			return true
		}
	}
	return false
}

// RemovePairsByFilter checks to see if a pair contains a specific currency
// and removes it from the list of pairs
func (p Pairs) RemovePairsByFilter(filter Code) Pairs {
	pairs := make(Pairs, 0, len(p))
	for i := range p {
		if p[i].Contains(filter) {
			continue
		}
		pairs = append(pairs, p[i])
	}
	return pairs
}

// GetPairsByFilter returns all pairs that have at least one match base or quote
// to the filter code.
func (p Pairs) GetPairsByFilter(filter Code) Pairs {
	pairs := make(Pairs, 0, len(p))
	for i := range p {
		if !p[i].Contains(filter) {
			continue
		}
		pairs = append(pairs, p[i])
	}
	return pairs
}

// GetPairsByCurrencies returns all pairs that have both matches to the
// currencies passed in. This allows for the construction of pairs by required
// currency codes.
func (p Pairs) GetPairsByCurrencies(currencies Currencies) Pairs {
	pairs := make(Pairs, 0, len(p))
	for i := range p {
		if currencies.Contains(p[i].Base) && currencies.Contains(p[i].Quote) {
			pairs = append(pairs, p[i])
		}
	}
	return pairs
}

// Remove removes the specified pair from the list of pairs if it exists
func (p Pairs) Remove(pair Pair) Pairs {
	pairs := make(Pairs, 0, len(p))
	for x := range p {
		if p[x].Equal(pair) {
			continue
		}
		pairs = append(pairs, p[x])
	}
	return pairs
}

// Add adds a specified pair to the list of pairs if it doesn't exist
func (p Pairs) Add(pair Pair) Pairs {
	if p.Contains(pair, true) {
		return p
	}
	p = append(p, pair)
	return p
}

// GetMatch returns either the pair that is equal including the reciprocal for
// when currencies are constructed from different exchange pairs e.g. Exchange
// one USDT-DAI to exchange two DAI-USDT enabled/available pairs.
func (p Pairs) GetMatch(pair Pair) (Pair, error) {
	for x := range p {
		if p[x].EqualIncludeReciprocal(pair) {
			return p[x], nil
		}
	}
	return EMPTYPAIR, ErrPairNotFound
}

// FindDifferences returns pairs which are new or have been removed
func (p Pairs) FindDifferences(pairs Pairs) (newPairs, removedPairs Pairs) {
	for x := range pairs {
		if pairs[x].String() == "" {
			continue
		}
		if !p.Contains(pairs[x], true) {
			newPairs = append(newPairs, pairs[x])
		}
	}
	for x := range p {
		if p[x].String() == "" {
			continue
		}
		if !pairs.Contains(p[x], true) {
			removedPairs = append(removedPairs, p[x])
		}
	}
	return
}

// GetRandomPair returns a random pair from a list of pairs
func (p Pairs) GetRandomPair() Pair {
	if pairsLen := len(p); pairsLen != 0 {
		return p[rand.Intn(pairsLen)] // nolint:gosec // basic number generation required, no need for crypo/rand
	}
	return EMPTYPAIR
}

// DeriveFrom matches symbol string to the available pairs list when no
// delimiter is supplied.
func (p Pairs) DeriveFrom(symbol string) (Pair, error) {
	if len(p) == 0 {
		return EMPTYPAIR, errPairsEmpty
	}
	if symbol == "" {
		return EMPTYPAIR, errSymbolEmpty
	}
	symbol = strings.ToLower(symbol)
pairs:
	for x := range p {
		if p[x].Len() != len(symbol) {
			continue
		}
		base := p[x].Base.Lower().String()
		baseLength := len(base)
		for y := 0; y < baseLength; y++ {
			if base[y] != symbol[y] {
				continue pairs
			}
		}
		quote := p[x].Quote.Lower().String()
		for y := 0; y < len(quote); y++ {
			if quote[y] != symbol[baseLength+y] {
				continue pairs
			}
		}
		return p[x], nil
	}
	return EMPTYPAIR, fmt.Errorf("%w for symbol string %s", ErrPairNotFound, symbol)
}

// GetCrypto returns all the cryptos contained in the list.
func (p Pairs) GetCrypto() Currencies {
	m := make(map[*Item]bool)
	for x := range p {
		if p[x].Base.IsCryptocurrency() {
			m[p[x].Base.Item] = p[x].Base.UpperCase
		}
		if p[x].Quote.IsCryptocurrency() {
			m[p[x].Quote.Item] = p[x].Quote.UpperCase
		}
	}
	return currencyConstructor(m)
}

// GetFiat returns all the cryptos contained in the list.
func (p Pairs) GetFiat() Currencies {
	m := make(map[*Item]bool)
	for x := range p {
		if p[x].Base.IsFiatCurrency() {
			m[p[x].Base.Item] = p[x].Base.UpperCase
		}
		if p[x].Quote.IsFiatCurrency() {
			m[p[x].Quote.Item] = p[x].Quote.UpperCase
		}
	}
	return currencyConstructor(m)
}

// GetCurrencies returns the full currency code list contained derived from the
// pairs list.
func (p Pairs) GetCurrencies() Currencies {
	m := make(map[*Item]bool)
	for x := range p {
		m[p[x].Base.Item] = p[x].Base.UpperCase
		m[p[x].Quote.Item] = p[x].Quote.UpperCase
	}
	return currencyConstructor(m)
}

// GetStables returns the stable currency code list derived from the pairs list.
func (p Pairs) GetStables() Currencies {
	m := make(map[*Item]bool)
	for x := range p {
		if p[x].Base.IsStableCurrency() {
			m[p[x].Base.Item] = p[x].Base.UpperCase
		}
		if p[x].Quote.IsStableCurrency() {
			m[p[x].Quote.Item] = p[x].Quote.UpperCase
		}
	}
	return currencyConstructor(m)
}

// currencyConstructor takes in an item map and returns the currencies with
// the same formatting.
func currencyConstructor(m map[*Item]bool) Currencies {
	var cryptos = make([]Code, len(m))
	var target int
	for code, upper := range m {
		cryptos[target].Item = code
		cryptos[target].UpperCase = upper
		target++
	}
	return cryptos
}

// GetStablesMatch returns all stable pairs matched with code
func (p Pairs) GetStablesMatch(code Code) Pairs {
	stablePairs := make([]Pair, 0, len(p))
	for x := range p {
		if p[x].Base.IsStableCurrency() && p[x].Quote.Equal(code) ||
			p[x].Quote.IsStableCurrency() && p[x].Base.Equal(code) {
			stablePairs = append(stablePairs, p[x])
		}
	}
	return stablePairs
}
