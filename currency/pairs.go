package currency

import (
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// Public Errors
var (
	ErrPairDuplication = errors.New("currency pair duplication")
)

var (
	errSymbolEmpty                = errors.New("symbol is empty")
	errNoDelimiter                = errors.New("no delimiter was supplied")
	errPairFormattingInconsistent = errors.New("pair formatting is inconsistent")
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
func (p Pairs) Format(pairFmt PairFormat) Pairs {
	pairs := slices.Clone(p)
	for x := range pairs {
		pairs[x].Base.upperCase = pairFmt.Uppercase
		pairs[x].Quote.upperCase = pairFmt.Uppercase
		pairs[x].Delimiter = pairFmt.Delimiter
	}
	return pairs
}

// UnmarshalJSON conforms type to the umarshaler interface
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

// Upper updates and returns the entire slice of pairs to upper casing
// NOTE: Do not duplicate slice reference as this can cause race issues.
func (p Pairs) Upper() Pairs {
	newSlice := make(Pairs, len(p))
	for i := range p {
		newSlice[i] = p[i].Upper()
	}
	return newSlice
}

// Lower updates and returns the entire slice of pairs to upper casing
// NOTE: Do not duplicate slice reference as this can cause race issues.
func (p Pairs) Lower() Pairs {
	newSlice := make(Pairs, len(p))
	for i := range p {
		newSlice[i] = p[i].Lower()
	}
	return newSlice
}

// Contains checks to see if a specified pair exists inside a currency pair array
func (p Pairs) Contains(check Pair, exact bool) bool {
	for i := range p {
		if (exact && p[i].Equal(check)) ||
			(!exact && p[i].EqualIncludeReciprocal(check)) {
			return true
		}
	}
	return false
}

// ContainsAll checks to see if all pairs supplied are contained within the original pairs list
func (p Pairs) ContainsAll(check Pairs, exact bool) error {
	if len(check) == 0 {
		return ErrCurrencyPairsEmpty
	}

	comparative := slices.Clone(p)
list:
	for x := range check {
		for y := range comparative {
			if (exact && check[x].Equal(comparative[y])) ||
				(!exact && check[x].EqualIncludeReciprocal(comparative[y])) {
				// Reduce list size to decrease array traversal speed on iteration.
				comparative[y] = comparative[len(comparative)-1]
				comparative = comparative[:len(comparative)-1]
				continue list
			}
		}

		// Opted for in error original check for duplication.
		if p.Contains(check[x], exact) {
			return fmt.Errorf("%s %w", check[x], ErrPairDuplication)
		}
		return fmt.Errorf("%s %w", check[x], ErrPairNotContainedInAvailablePairs)
	}
	return nil
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

// Remove removes the specified pairs from the list of pairs if they exist
func (p Pairs) Remove(rem ...Pair) Pairs {
	n := make(Pairs, 0, len(p))
	for _, pN := range p {
		if !slices.ContainsFunc(rem, func(pX Pair) bool { return pX.Equal(pN) }) {
			n = append(n, pN)
		}
	}
	return slices.Clip(n)
}

// Add adds pairs to the list of pairs ignoring duplicates
func (p Pairs) Add(pairs ...Pair) Pairs {
	n := slices.Clone(p)
	for _, a := range pairs {
		if !n.Contains(a, true) {
			n = append(n, a)
		}
	}
	return n
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

type pairKey struct {
	Base  *Item
	Quote *Item
}

// FindDifferences returns pairs which are new or have been removed
func (p Pairs) FindDifferences(incoming Pairs, pairFmt PairFormat) (PairDifference, error) {
	newPairs := make(Pairs, 0, len(incoming))
	check := make(map[pairKey]bool)
	formatDiff := false
	for x := range incoming {
		if incoming[x].IsEmpty() {
			return PairDifference{}, fmt.Errorf("contained in the incoming pairs a %w", ErrCurrencyPairEmpty)
		}

		if !formatDiff {
			formatDiff = incoming[x].hasFormatDifference(pairFmt)
		}

		k := pairKey{Base: incoming[x].Base.Item, Quote: incoming[x].Quote.Item}
		if check[k] {
			return PairDifference{}, fmt.Errorf("contained in the incoming pairs %w", ErrPairDuplication)
		}
		check[k] = true
		if !p.Contains(incoming[x], true) {
			newPairs = append(newPairs, incoming[x])
		}
	}
	removedPairs := make(Pairs, 0, len(p))
	clear(check)
	for x := range p {
		if p[x].IsEmpty() {
			return PairDifference{}, fmt.Errorf("contained in the existing pairs a %w", ErrCurrencyPairEmpty)
		}

		if !formatDiff {
			formatDiff = p[x].hasFormatDifference(pairFmt)
		}

		k := pairKey{Base: p[x].Base.Item, Quote: p[x].Quote.Item}
		if !incoming.Contains(p[x], true) || check[k] {
			removedPairs = append(removedPairs, p[x])
		}
		check[k] = true
	}
	return PairDifference{New: newPairs, Remove: removedPairs, FormatDifference: formatDiff}, nil
}

// HasFormatDifference checks and validates full formatting across a pairs list
func (p Pairs) HasFormatDifference(pairFmt PairFormat) bool {
	return slices.ContainsFunc(p, func(pair Pair) bool { return pair.hasFormatDifference(pairFmt) })
}

// GetRandomPair returns a random pair from a list of pairs
func (p Pairs) GetRandomPair() (Pair, error) {
	if len(p) == 0 {
		return EMPTYPAIR, ErrCurrencyPairsEmpty
	}
	return p[rand.Intn(len(p))], nil //nolint:gosec // basic number generation required, no need for crypo/rand
}

// DeriveFrom matches symbol string to the available pairs list when no
// delimiter is supplied. WARNING: This is not optimised and should only be used
// for one off processes.
func (p Pairs) DeriveFrom(symbol string) (Pair, error) {
	if len(p) == 0 {
		return EMPTYPAIR, ErrCurrencyPairsEmpty
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
		for y := range baseLength {
			if base[y] != symbol[y] {
				continue pairs
			}
		}
		quote := p[x].Quote.Lower().String()
		for y := range quote {
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
			m[p[x].Base.Item] = p[x].Base.upperCase
		}
		if p[x].Quote.IsCryptocurrency() {
			m[p[x].Quote.Item] = p[x].Quote.upperCase
		}
	}
	return currencyConstructor(m)
}

// GetFiat returns all the cryptos contained in the list.
func (p Pairs) GetFiat() Currencies {
	m := make(map[*Item]bool)
	for x := range p {
		if p[x].Base.IsFiatCurrency() {
			m[p[x].Base.Item] = p[x].Base.upperCase
		}
		if p[x].Quote.IsFiatCurrency() {
			m[p[x].Quote.Item] = p[x].Quote.upperCase
		}
	}
	return currencyConstructor(m)
}

// GetCurrencies returns the full currency code list contained derived from the
// pairs list.
func (p Pairs) GetCurrencies() Currencies {
	m := make(map[*Item]bool)
	for x := range p {
		m[p[x].Base.Item] = p[x].Base.upperCase
		m[p[x].Quote.Item] = p[x].Quote.upperCase
	}
	return currencyConstructor(m)
}

// GetStables returns the stable currency code list derived from the pairs list.
func (p Pairs) GetStables() Currencies {
	m := make(map[*Item]bool)
	for x := range p {
		if p[x].Base.IsStableCurrency() {
			m[p[x].Base.Item] = p[x].Base.upperCase
		}
		if p[x].Quote.IsStableCurrency() {
			m[p[x].Quote.Item] = p[x].Quote.upperCase
		}
	}
	return currencyConstructor(m)
}

// currencyConstructor takes in an item map and returns the currencies with
// the same formatting.
func currencyConstructor(m map[*Item]bool) Currencies {
	cryptos := make([]Code, len(m))
	var target int
	for code, upper := range m {
		cryptos[target].Item = code
		cryptos[target].upperCase = upper
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

// ValidateAndConform checks for duplications and empty pairs then conforms the
// entire pairs list to the supplied formatting (unless bypassed).
// Map[string]bool type is used to make sure delimiters are not included so
// different formatting entry duplications can be found e.g. `LINKUSDTM21`,
// `LIN-KUSDTM21` or `LINK-USDTM21 are all the same instances but with different
// unintentional processes for formatting.
func (p Pairs) ValidateAndConform(pFmt PairFormat, bypassFormatting bool) (Pairs, error) {
	processedPairs := make(map[string]bool, len(p))
	formatted := make(Pairs, len(p))
	var target int
	for x := range p {
		if p[x].IsEmpty() {
			return nil, fmt.Errorf("cannot update pairs %w", ErrCurrencyPairEmpty)
		}
		strippedPair := EMPTYFORMAT.Format(p[x])
		if processedPairs[strippedPair] {
			return nil, fmt.Errorf("cannot update pairs %w with [%s]", ErrPairDuplication, p[x])
		}
		// Force application of supplied formatting
		processedPairs[strippedPair] = true
		if !bypassFormatting {
			formatted[target] = p[x].Format(pFmt)
		} else {
			formatted[target] = p[x]
		}
		target++
	}
	return formatted, nil
}

// GetFormatting returns the formatting of a set of pairs
func (p Pairs) GetFormatting() (PairFormat, error) {
	if len(p) == 0 {
		return PairFormat{}, ErrCurrencyPairsEmpty
	}
	pFmt, err := p[0].GetFormatting()
	if err != nil {
		return PairFormat{}, err
	}
	if p.HasFormatDifference(pFmt) {
		return PairFormat{}, errPairFormattingInconsistent
	}
	return pFmt, nil
}

// GetPairsByQuote returns all pairs that have a matching quote currency
func (p Pairs) GetPairsByQuote(quoteTerm Code) (Pairs, error) {
	if len(p) == 0 {
		return nil, ErrCurrencyPairsEmpty
	}
	if quoteTerm.IsEmpty() {
		return nil, ErrCurrencyCodeEmpty
	}
	pairs := make(Pairs, 0, len(p))
	for i := range p {
		if p[i].Quote.Equal(quoteTerm) {
			pairs = append(pairs, p[i])
		}
	}
	return pairs, nil
}

// GetPairsByBase returns all pairs that have a matching base currency
func (p Pairs) GetPairsByBase(baseTerm Code) (Pairs, error) {
	if len(p) == 0 {
		return nil, ErrCurrencyPairsEmpty
	}
	if baseTerm.IsEmpty() {
		return nil, ErrCurrencyCodeEmpty
	}
	pairs := make(Pairs, 0, len(p))
	for i := range p {
		if p[i].Base.Equal(baseTerm) {
			pairs = append(pairs, p[i])
		}
	}
	return pairs, nil
}

// equalKey is a small key for testing pair equality without delimiter
type equalKey struct {
	Base  *Item
	Quote *Item
}

// Equal checks to see if two lists of pairs contain only the same pairs, ignoring delimiter and case
// Does not check for inverted/reciprocal pairs
func (p Pairs) Equal(b Pairs) bool {
	if len(p) != len(b) {
		return false
	}
	if len(p) == 0 {
		return true
	}
	m := map[equalKey]struct{}{}
	for i := range p {
		m[equalKey{Base: p[i].Base.Item, Quote: p[i].Quote.Item}] = struct{}{}
	}
	for i := range b {
		if _, ok := m[equalKey{Base: b[i].Base.Item, Quote: b[i].Quote.Item}]; !ok {
			return false
		}
	}
	return true
}
