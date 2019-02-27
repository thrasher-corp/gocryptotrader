package currency

import (
	"math/rand"

	"github.com/thrasher-/gocryptotrader/common"
)

// Pairs defines a list of pairs
type Pairs []Pair

// Strings returns a slice of strings refering to each currency pair
func (p Pairs) Strings() []string {
	var list []string
	for _, pair := range p {
		list = append(list, pair.String())
	}
	return list
}

// Join returns a comma separated list of currency pairs
func (p Pairs) Join() string {
	return common.JoinStrings(p.Strings(), ",")
}

// Format formats the pair list to the exchange format configuration
func (p Pairs) Format(delimiter, index string, uppercase bool) Pairs {
	var pairs Pairs
	for _, data := range p {
		var formattedPair Pair
		formattedPair.Delimiter = delimiter
		formattedPair.Base = data.Base
		formattedPair.Quote = data.Quote

		if index != "" {
			formattedPair.Quote = NewCode(index)
		}

		if uppercase {
			pairs = append(pairs, formattedPair.Upper())
		} else {
			pairs = append(pairs, formattedPair)
		}
	}
	return pairs
}

// UnmarshalJSON comforms type to the umarshaler interface
func (p *Pairs) UnmarshalJSON(d []byte) error {
	var pairs string
	err := common.JSONDecode(d, &pairs)
	if err != nil {
		return err
	}

	var allThePairs Pairs
	for _, data := range common.SplitStrings(pairs, ",") {
		allThePairs = append(allThePairs, NewPairFromString(data))
	}

	*p = allThePairs
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (p Pairs) MarshalJSON() ([]byte, error) {
	return common.JSONEncode(p.Join())
}

// Upper returns an upper formatted pair list
func (p Pairs) Upper() Pairs {
	var upper Pairs
	for _, data := range p {
		upper = append(upper, data.Upper())
	}
	return upper
}

// Slice exposes the underlying type
func (p Pairs) Slice() []Pair {
	return p
}

// Contain checks to see if a specified pair exists inside a currency pair
// array
func (p Pairs) Contain(check Pair, exact bool) bool {
	for _, pair := range p.Slice() {
		if exact {
			if pair.Equal(check) {
				return true
			}
		} else {
			if pair.EqualIncludeReciprocal(check) {
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
	for _, pair := range p.Slice() {
		if pair.ContainsCurrency(filter) {
			continue
		}
		pairs = append(pairs, pair)
	}
	return pairs
}

// FindDifferences returns pairs which are new or have been removed
func (p Pairs) FindDifferences(newPairs Pairs) (Pairs, Pairs) {
	var newPs, removedPs Pairs
	for x := range newPairs {
		if newPairs[x].String() == "" {
			continue
		}
		if !p.Contain(newPairs[x], true) {
			newPs = append(newPs, newPairs[x])
		}
	}
	for _, oldPair := range p.Slice() {
		if oldPair.String() == "" {
			continue
		}
		if !newPairs.Contain(oldPair, true) {
			removedPs = append(removedPs, oldPair)
		}
	}
	return newPs, removedPs
}

// GetRandomPair returns a random pair from a list of pairs
func (p Pairs) GetRandomPair() Pair {
	pairsLen := len(p)

	if pairsLen == 0 {
		return Pair{Base: NewCode(""), Quote: NewCode("")}
	}

	return p[rand.Intn(pairsLen)]
}
