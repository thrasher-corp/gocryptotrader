package currency

import (
	"math/rand"
	"strings"

	"github.com/thrasher-/gocryptotrader/common"
)

// Pairs defines a list of pairs
type Pairs []Pair

// String returns a slice of strings refering to each currency pair
func (p Pairs) String() []string {
	var list []string
	for _, pair := range p {
		list = append(list, pair.String())
	}
	return list
}

// Join returns a comma separated list of currency pairs
func (p Pairs) Join() string {
	return common.JoinStrings(p.String(), ",")
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
			formattedPair.Quote = Code(index)
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
		allThePairs = append(allThePairs, NewCurrencyPairFromString(data))
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

	*p = NewCurrencyPairFromString(pair)
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (p Pair) MarshalJSON() ([]byte, error) {
	return common.JSONEncode(p.String())
}

// Display formats and returns the currency based on user preferences,
// overriding the default String() display
func (p Pair) Display(delimiter string, uppercase bool) Pair {
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

// Swap turns the currency pair into its reciprocal
func (p Pair) Swap() Pair {
	b := p.Base
	p.Base = p.Quote
	p.Quote = b
	return p
}

// Empty returns whether or not the pair is empty or is missing a currency code
func (p Pair) Empty() bool {
	if p.Base == "" || p.Quote == "" {
		return true
	}
	return false
}

// NewCurrencyPairListFromString takes in currency pair strings and returns a
// currency pair list
func NewCurrencyPairListFromString(pairs []string) Pairs {
	var ps Pairs
	for _, p := range pairs {
		if p == "" {
			continue
		}

		ps = append(ps, NewCurrencyPairFromString(p))
	}
	return ps
}

// NewCurrencyPairDelimiter splits the desired currency string at delimeter,
// the returns a CurrencyPair struct
func NewCurrencyPairDelimiter(currencyPair, delimiter string) Pair {
	result := strings.Split(currencyPair, delimiter)
	return Pair{
		Delimiter: delimiter,
		Base:      Code(result[0]),
		Quote:     Code(result[1]),
	}
}

// NewCurrencyPair returns a CurrencyPair without a delimiter
func NewCurrencyPair(BaseCurrency, QuoteCurrency Code) Pair {
	return Pair{
		Base:  BaseCurrency,
		Quote: QuoteCurrency,
	}
}

// NewCurrencyPairWithDelimiter returns a CurrencyPair with a delimiter
func NewCurrencyPairWithDelimiter(base, quote, delimiter string) Pair {
	return Pair{
		Base:      Code(base),
		Quote:     Code(quote),
		Delimiter: delimiter,
	}
}

// NewCurrencyPairFromIndex returns a CurrencyPair via a currency string and
// specific index
func NewCurrencyPairFromIndex(currencyPair, index string) Pair {
	i := strings.Index(currencyPair, index)
	if i == 0 {
		return NewCurrencyPair(Code(currencyPair[0:len(index)]),
			Code(currencyPair[len(index):]))
	}
	return NewCurrencyPair(Code(currencyPair[0:i]), Code(currencyPair[i:]))
}

// NewCurrencyPairFromString converts currency string into a new CurrencyPair
// with or without delimeter
func NewCurrencyPairFromString(currencyPair string) Pair {
	delimiters := []string{"_", "-"}
	var delimiter string
	for _, x := range delimiters {
		if strings.Contains(currencyPair, x) {
			delimiter = x
			return NewCurrencyPairDelimiter(currencyPair, delimiter)
		}
	}
	return NewCurrencyPair(Code(currencyPair[0:3]), Code(currencyPair[3:]))
}

// PairsContain checks to see if a specified pair exists inside a currency pair
// array
func PairsContain(pairs []Pair, p Pair, exact bool) bool {
	for x := range pairs {
		if exact {
			if pairs[x].Equal(p) {
				return true
			}
		}
		if pairs[x].EqualIncludeReciprocal(p) {
			return true
		}
	}
	return false
}

// ContainsCurrency checks to see if a pair contains a specific currency
func ContainsCurrency(p Pair, c string) bool {
	return p.Base.Upper().String() == common.StringToUpper(c) ||
		p.Quote.Upper().String() == common.StringToUpper(c)
}

// RemovePairsByFilter checks to see if a pair contains a specific currency
// and removes it from the list of pairs
func RemovePairsByFilter(p []Pair, filter string) []Pair {
	var pairs []Pair
	for x := range p {
		if ContainsCurrency(p[x], filter) {
			continue
		}
		pairs = append(pairs, p[x])
	}
	return pairs
}

// FormatPairs formats a string array to a list of currency pairs with the
// supplied currency pair format
func FormatPairs(pairs []string, delimiter, index string) []Pair {
	var result []Pair
	for x := range pairs {
		if pairs[x] == "" {
			continue
		}
		var p Pair
		if delimiter != "" {
			p = NewCurrencyPairDelimiter(pairs[x], delimiter)
		} else {
			if index != "" {
				p = NewCurrencyPairFromIndex(pairs[x], index)
			} else {
				p = NewCurrencyPair(Code(pairs[x][0:3]), Code(pairs[x][3:]))
			}
		}
		result = append(result, p)
	}
	return result
}

// CopyPairFormat copies the pair format from a list of pairs once matched
func CopyPairFormat(p Pair, pairs []Pair, exact bool) Pair {
	for x := range pairs {
		if exact {
			if p.Equal(pairs[x]) {
				return pairs[x]
			}
		}
		if p.EqualIncludeReciprocal(pairs[x]) {
			return pairs[x]
		}
	}
	return Pair{}
}

// FindPairDifferences returns pairs which are new or have been removed
func FindPairDifferences(oldPairs, newPairs Pairs) (Pairs, Pairs) {
	var newPs, removedPs Pairs
	for x := range newPairs {
		if newPairs[x].String() == "" {
			continue
		}
		if !common.StringDataCompareUpper(oldPairs.String(),
			newPairs[x].String()) {
			newPs = append(newPs, newPairs[x])
		}
	}
	for x := range oldPairs {
		if oldPairs[x].String() == "" {
			continue
		}
		if !common.StringDataCompareUpper(newPairs.String(),
			oldPairs[x].String()) {
			removedPs = append(removedPs, oldPairs[x])
		}
	}
	return newPs, removedPs
}

// PairsToStringArray returns a list of pairs as a string array
func PairsToStringArray(pairs []Pair) []string {
	var p []string
	for x := range pairs {
		p = append(p, pairs[x].String())
	}
	return p
}

// RandomPairFromPairs returns a random pair from a list of pairs
func RandomPairFromPairs(pairs Pairs) Pair {
	pairsLen := len(pairs)

	if pairsLen == 0 {
		return Pair{}
	}

	return pairs[rand.Intn(pairsLen)]
}
