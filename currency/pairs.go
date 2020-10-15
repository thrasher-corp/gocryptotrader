package currency

import (
	"encoding/json"
	"math/rand"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/log"
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

	var allThePairs Pairs
	oldPairs := strings.Split(pairs, ",")
	for i := range oldPairs {
		pair, err := NewPairFromString(oldPairs[i])
		if err != nil {
			return err
		}
		allThePairs = append(allThePairs, pair)
	}

	*p = allThePairs
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (p Pairs) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Join())
}

// Upper returns an upper formatted pair list
func (p Pairs) Upper() Pairs {
	var upper Pairs
	for i := range p {
		upper = append(upper, p[i].Upper())
	}
	return upper
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
	pairsLen := len(p)

	if pairsLen == 0 {
		return Pair{Base: NewCode(""), Quote: NewCode("")}
	}

	return p[rand.Intn(pairsLen)] // nolint:gosec // basic number generation required, no need for crypo/rand
}
