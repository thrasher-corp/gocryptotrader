package currency

import (
	"strings"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

// NewCurrenciesFromStringArray returns a Currencies object from strings
func NewCurrenciesFromStringArray(currencies []string) Currencies {
	list := make(Currencies, 0, len(currencies))
	for i := range currencies {
		if currencies[i] == "" {
			continue
		}
		list = append(list, NewCode(currencies[i]))
	}
	return list
}

// Currencies define a range of supported currency codes
type Currencies []Code

// Add adds a currency to the list if it doesn't exist
func (c Currencies) Add(a Code) Currencies {
	if !c.Contains(a) {
		c = append(c, a)
	}
	return c
}

// Strings returns an array of currency strings
func (c Currencies) Strings() []string {
	list := make([]string, len(c))
	for i := range c {
		list[i] = c[i].String()
	}
	return list
}

// Contains checks to see if a currency code is contained in the currency list
func (c Currencies) Contains(check Code) bool {
	for i := range c {
		if c[i].Equal(check) {
			return true
		}
	}
	return false
}

// Join returns a comma serparated string
func (c Currencies) Join() string {
	return strings.Join(c.Strings(), ",")
}

// UnmarshalJSON conforms type to the umarshaler interface
func (c *Currencies) UnmarshalJSON(d []byte) error {
	if d[0] != '[' {
		d = []byte(`[` + string(d) + `]`)
	}
	return json.Unmarshal(d, (*[]Code)(c))
}

// MarshalJSON conforms type to the marshaler interface
func (c Currencies) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Join())
}

// Match returns if the full list equals the supplied list
func (c Currencies) Match(other Currencies) bool {
	if len(c) != len(other) {
		return false
	}

match:
	for x := range c {
		for y := range other {
			if c[x].Equal(other[y]) {
				continue match
			}
		}
		return false
	}
	return true
}

// HasData checks to see if Currencies type has actual currencies
func (c Currencies) HasData() bool {
	return len(c) != 0
}
