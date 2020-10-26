package currency

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

func (r Role) String() string {
	switch r {
	case Unset:
		return UnsetRoleString
	case Fiat:
		return FiatCurrencyString
	case Cryptocurrency:
		return CryptocurrencyString
	case Token:
		return TokenString
	case Contract:
		return ContractString
	default:
		return "UNKNOWN"
	}
}

// MarshalJSON conforms Role to the marshaller interface
func (r Role) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

// UnmarshalJSON conforms Role to the unmarshaller interface
func (r *Role) UnmarshalJSON(d []byte) error {
	var incoming string
	err := json.Unmarshal(d, &incoming)
	if err != nil {
		return err
	}

	switch incoming {
	case UnsetRoleString:
		*r = Unset
	case FiatCurrencyString:
		*r = Fiat
	case CryptocurrencyString:
		*r = Cryptocurrency
	case TokenString:
		*r = Token
	case ContractString:
		*r = Contract
	default:
		return fmt.Errorf("unmarshal error role type %s unsupported for currency",
			incoming)
	}
	return nil
}

// HasData returns true if the type contains data
func (b *BaseCodes) HasData() bool {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	return len(b.Items) != 0
}

// GetFullCurrencyData returns a type that is read to dump to file
func (b *BaseCodes) GetFullCurrencyData() (File, error) {
	var file File
	for i := range b.Items {
		switch b.Items[i].Role {
		case Unset:
			file.UnsetCurrency = append(file.UnsetCurrency, *b.Items[i])
		case Fiat:
			file.FiatCurrency = append(file.FiatCurrency, *b.Items[i])
		case Cryptocurrency:
			file.Cryptocurrency = append(file.Cryptocurrency, *b.Items[i])
		case Token:
			file.Token = append(file.Token, *b.Items[i])
		case Contract:
			file.Contracts = append(file.Contracts, *b.Items[i])
		default:
			return file, errors.New("role undefined")
		}
	}

	file.LastMainUpdate = b.LastMainUpdate.Unix()
	return file, nil
}

// GetCurrencies gets the full currency list from the base code type available
// from the currency system
func (b *BaseCodes) GetCurrencies() Currencies {
	var currencies Currencies
	b.mtx.Lock()
	for i := range b.Items {
		currencies = append(currencies, Code{
			Item: b.Items[i],
		})
	}
	b.mtx.Unlock()
	return currencies
}

// UpdateCurrency updates or registers a currency/contract
func (b *BaseCodes) UpdateCurrency(fullName, symbol, blockchain string, id int, r Role) error {
	if r == Unset {
		return fmt.Errorf("role cannot be unset in update currency for %s", symbol)
	}
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol != symbol {
			continue
		}

		if b.Items[i].Role == Unset {
			b.Items[i].FullName = fullName
			b.Items[i].Role = r
			b.Items[i].AssocChain = blockchain
			b.Items[i].ID = id
			return nil
		}

		if b.Items[i].Role != r {
			// Captures same name currencies and duplicates to different roles
			break
		}

		b.Items[i].FullName = fullName
		b.Items[i].AssocChain = blockchain
		b.Items[i].ID = id
		return nil
	}

	b.Items = append(b.Items, &Item{
		Symbol:     symbol,
		FullName:   fullName,
		Role:       r,
		AssocChain: blockchain,
		ID:         id,
	})
	return nil
}

// Register registers a currency from a string and returns a currency code
func (b *BaseCodes) Register(c string) Code {
	var format bool
	if c != "" {
		format = unicode.IsUpper(rune(c[0]))
	}
	// Force upper string storage and matching
	c = strings.ToUpper(c)

	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol == c {
			return Code{
				Item:      b.Items[i],
				UpperCase: format,
			}
		}
	}

	newItem := &Item{Symbol: c}
	b.Items = append(b.Items, newItem)

	return Code{
		Item:      newItem,
		UpperCase: format,
	}
}

// RegisterFiat registers a fiat currency from a string and returns a currency
// code
func (b *BaseCodes) RegisterFiat(c string) Code {
	c = strings.ToUpper(c)

	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol == c {
			if b.Items[i].Role == Unset {
				b.Items[i].Role = Fiat
			}

			if b.Items[i].Role != Fiat {
				continue
			}
			return Code{Item: b.Items[i], UpperCase: true}
		}
	}

	item := &Item{Symbol: c, Role: Fiat}
	b.Items = append(b.Items, item)
	return Code{Item: item, UpperCase: true}
}

// LoadItem sets item data
func (b *BaseCodes) LoadItem(item *Item) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol != item.Symbol ||
			(b.Items[i].Role != Unset &&
				item.Role != Unset &&
				b.Items[i].Role != item.Role) {
			continue
		}
		b.Items[i].AssocChain = item.AssocChain
		b.Items[i].ID = item.ID
		b.Items[i].Role = item.Role
		b.Items[i].FullName = item.FullName
		return nil
	}
	b.Items = append(b.Items, item)
	return nil
}

// NewCode returns a new currency registered code
func NewCode(c string) Code {
	return storage.ValidateCode(c)
}

// String conforms to the stringer interface
func (i *Item) String() string {
	return i.FullName
}

// String converts the code to string
func (c Code) String() string {
	if c.Item == nil {
		return ""
	}

	if c.UpperCase {
		return strings.ToUpper(c.Item.Symbol)
	}
	return strings.ToLower(c.Item.Symbol)
}

// Lower converts the code to lowercase formatting
func (c Code) Lower() Code {
	c.UpperCase = false
	return c
}

// Upper converts the code to uppercase formatting
func (c Code) Upper() Code {
	c.UpperCase = true
	return c
}

// UnmarshalJSON comforms type to the umarshaler interface
func (c *Code) UnmarshalJSON(d []byte) error {
	var newcode string
	err := json.Unmarshal(d, &newcode)
	if err != nil {
		return err
	}
	*c = NewCode(newcode)
	return nil
}

// MarshalJSON conforms type to the marshaler interface
func (c Code) MarshalJSON() ([]byte, error) {
	if c.Item == nil {
		return json.Marshal("")
	}
	return json.Marshal(c.String())
}

// IsEmpty returns true if the code is empty
func (c Code) IsEmpty() bool {
	if c.Item == nil {
		return true
	}
	return c.Item.Symbol == ""
}

// Match returns if the code supplied is the same as the corresponding code
func (c Code) Match(check Code) bool {
	return c.Item == check.Item
}

// IsDefaultFiatCurrency checks if the currency passed in matches the default
// fiat currency
func (c Code) IsDefaultFiatCurrency() bool {
	return storage.IsDefaultCurrency(c)
}

// IsDefaultCryptocurrency checks if the currency passed in matches the default
// cryptocurrency
func (c Code) IsDefaultCryptocurrency() bool {
	return storage.IsDefaultCryptocurrency(c)
}

// IsFiatCurrency checks if the currency passed is an enabled fiat currency
func (c Code) IsFiatCurrency() bool {
	return storage.IsFiatCurrency(c)
}

// IsCryptocurrency checks if the currency passed is an enabled CRYPTO currency.
func (c Code) IsCryptocurrency() bool {
	return storage.IsCryptocurrency(c)
}
