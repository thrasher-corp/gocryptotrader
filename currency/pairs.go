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
)

// NewPairsFromStrings takes in currency pair strings and returns a currency
// pair list
func NewPairsFromStrings(pairs []string) (Pairs, error) {
	var newPairs Pairs
	for i := range pairs {
		if pairs[i] == "" {
			continue
		}

		newPair, err := NewPairFromString(pairs[i])
		if err != nil {
			return nil, err
		}

		newPairs = append(newPairs, newPair)
	}
	return newPairs, nil
}

// NewPairsFromString takes in a comma delimitered string and returns a Pairs
// type
func NewPairsFromString(pairs string) (Pairs, error) {
	pairsSplit := strings.Split(pairs, ",")
	allThePairs := make(Pairs, len(pairsSplit))
	var err error
	for i := range pairsSplit {
		allThePairs[i], err = NewPairFromString(pairsSplit[i])
		if err != nil {
			return nil, err
		}
	}
	return allThePairs, nil
}

// Strings returns a slice of strings referring to each currency pair
func (p Pairs) Strings() []string {
	var list []string
	for i := range p {
		list = append(list, p[i].String())
	}
	return list
}

// Join returns a comma separated list of currency pairs
func (p Pairs) Join() string {
	return strings.Join(p.Strings(), ",")
}

// Format formats the pair list to the exchange format configuration
func (p Pairs) Format(delimiter, index string, uppercase bool) Pairs {
	var pairs Pairs
	for i := range p {
		var formattedPair = Pair{
			Delimiter: delimiter,
			Base:      p[i].Base,
			Quote:     p[i].Quote,
		}
		if index != "" {
			newP, err := NewPairFromIndex(p[i].String(), index)
			if err != nil {
				log.Errorf(log.Global,
					"failed to create NewPairFromIndex. Err: %s\n", err)
				continue
			}
			formattedPair.Base = newP.Base
			formattedPair.Quote = newP.Quote
		}

		if uppercase {
			pairs = append(pairs, formattedPair.Upper())
		} else {
			pairs = append(pairs, formattedPair.Lower())
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

	*p, err = NewPairsFromString(pairs)
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

// RemovePairsByFilter checks to see if a pair contains a specific currency
// and removes it from the list of pairs
func (p Pairs) RemovePairsByFilter(filter Code) Pairs {
	var pairs Pairs
	for i := range p {
		if p[i].ContainsCurrency(filter) {
			continue
		}
		pairs = append(pairs, p[i])
	}
	return pairs
}

// GetPairsByFilter returns all pairs that have at least one match base or quote
// to the filter code.
func (p Pairs) GetPairsByFilter(filter Code) Pairs {
	var pairs Pairs
	for i := range p {
		if !p[i].ContainsCurrency(filter) {
			continue
		}
		pairs = append(pairs, p[i])
	}
	return pairs
}

// Remove removes the specified pair from the list of pairs if it exists
func (p Pairs) Remove(pair Pair) Pairs {
	var pairs Pairs
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
	return Pair{}
}

// DerivePairFrom is able to match the incoming string without a delimiter
// against all contained pairs
func (p Pairs) DerivePairFrom(symbol string) (Pair, error) {
	if len(p) == 0 {
		return Pair{}, errPairsEmpty
	}
	if symbol == "" {
		return Pair{}, errSymbolEmpty
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
	return Pair{}, fmt.Errorf("%w for symbol string %s", ErrPairNotFound, symbol)
}

// GetCrypto returns all the crypto currencies contained in the list.
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
	var cryptos = make([]Code, len(m))
	var target int
	for code, upper := range m {
		cryptos[target].Item = code
		cryptos[target].UpperCase = upper
		target++
	}
	return cryptos
}

// GetFiat returns all the the fiat currencies contained in the list.
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
	var fiat = make([]Code, len(m))
	var target int
	for code, upper := range m {
		fiat[target].Item = code
		fiat[target].UpperCase = upper
		target++
	}
	return fiat
}

// GetCurrencies returns the full currency code list contained derived from the
// the pairs list.
func (p Pairs) GetCurrencies() Currencies {
	m := make(map[*Item]bool)
	for x := range p {
		m[p[x].Base.Item] = p[x].Base.UpperCase
		m[p[x].Quote.Item] = p[x].Quote.UpperCase
	}
	var currencies = make([]Code, len(m))
	var target int
	for code, upper := range m {
		currencies[target].Item = code
		currencies[target].UpperCase = upper
		target++
	}
	return currencies
}
