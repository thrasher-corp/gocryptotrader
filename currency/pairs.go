package currency

import (
	"math/rand"
	"strings"

	"github.com/thrasher-/gocryptotrader/common"
	log "github.com/thrasher-/gocryptotrader/logger"
)

// NewPairsFromStrings takes in currency pair strings and returns a currency
// pair list
func NewPairsFromStrings(pairs []string) Pairs {
	var ps Pairs
	for _, p := range pairs {
		if p == "" {
			continue
		}

		ps = append(ps, NewPairFromString(p))
	}
	return ps
}

// Pairs defines a list of pairs
type Pairs []Pair

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
				log.Errorf(log.LogGlobal,
					"failed to create NewPairFromIndex. Err: %s", err)
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
	err := common.JSONDecode(d, &pairs)
	if err != nil {
		return err
	}

	var allThePairs Pairs
	for _, data := range strings.Split(pairs, ",") {
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
	for i := range p {
		upper = append(upper, p[i].Upper())
	}
	return upper
}

// Slice exposes the underlying type
func (p Pairs) Slice() []Pair {
	return p
}

// Contains checks to see if a specified pair exists inside a currency pair
// array
func (p Pairs) Contains(check Pair, exact bool) bool {
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
func (p Pairs) FindDifferences(pairs Pairs) (newPairs, removedPairs Pairs) {
	for x := range pairs {
		if pairs[x].String() == "" {
			continue
		}
		if !p.Contains(pairs[x], true) {
			newPairs = append(newPairs, pairs[x])
		}
	}
	for _, oldPair := range p {
		if oldPair.String() == "" {
			continue
		}
		if !pairs.Contains(oldPair, true) {
			removedPairs = append(removedPairs, oldPair)
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

	return p[rand.Intn(pairsLen)]
}
