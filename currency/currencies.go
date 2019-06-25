package currency

import "github.com/idoall/gocryptotrader/common"

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
	for _, d := range c {
		list = append(list, d.String())
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
	return common.JoinStrings(c.Strings(), ",")
}

// UnmarshalJSON comforms type to the umarshaler interface
func (c *Currencies) UnmarshalJSON(d []byte) error {
	var configCurrencies string
	err := common.JSONDecode(d, &configCurrencies)
	if err != nil {
		return err
	}

	var allTheCurrencies Currencies
	for _, data := range common.SplitStrings(configCurrencies, ",") {
		allTheCurrencies = append(allTheCurrencies, NewCode(data))
	}

	*c = allTheCurrencies
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (c Currencies) MarshalJSON() ([]byte, error) {
	return common.JSONEncode(c.Join())
}

// Match returns if the full list equals the supplied list
func (c Currencies) Match(other Currencies) bool {
	if len(c) != len(other) {
		return false
	}

	for _, d := range c {
		var found bool
		for i := range other {
			if d == other[i] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// Slice exposes the underlying type
func (c Currencies) Slice() []Code {
	return c
}

// HasData checks to see if Currencies type has actual currencies
func (c Currencies) HasData() bool {
	return len(c) != 0
}
