package currency

import (
	"encoding/json"
	"strings"
)

// NewCurrenciesFromStringArray returns a Currencies object from strings
func NewCurrenciesFromStringArray(currencies []string) Currencies {
	var list Currencies
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

// Strings returns an array of currency strings
func (c Currencies) Strings() []string {
	var list []string
	for i := range c {
		list = append(list, c[i].String())
	}
	return list
}

// Contains checks to see if a currency code is contained in the currency list
func (c Currencies) Contains(cc Code) bool {
	for i := range c {
		if c[i].Item == cc.Item {
			return true
		}
	}
	return false
}

// Join returns a comma serparated string
func (c Currencies) Join() string {
	return strings.Join(c.Strings(), ",")
}

// UnmarshalJSON comforms type to the umarshaler interface
func (c *Currencies) UnmarshalJSON(d []byte) error {
	var configCurrencies string
	err := json.Unmarshal(d, &configCurrencies)
	if err != nil {
		return err
	}

	var allTheCurrencies Currencies
	curr := strings.Split(configCurrencies, ",")
	for i := range curr {
		allTheCurrencies = append(allTheCurrencies, NewCode(curr[i]))
	}

	*c = allTheCurrencies
	return nil
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
			if c[x] == other[y] {
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
