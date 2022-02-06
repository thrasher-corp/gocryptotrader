package currency

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
)

var (
	// ErrCurrencyCodeEmpty defines an error if the currency code is empty
	ErrCurrencyCodeEmpty = errors.New("currency code is empty")
	errItemIsNil         = errors.New("item is nil")
	errItemIsEmpty       = errors.New("item is empty")
	errRoleUnset         = errors.New("role unset")

	// EMPTY is an empty currency code
	EMPTY = Code{}
)

func (r Role) String() string {
	switch r {
	case Fiat:
		return FiatCurrencyString
	case Cryptocurrency:
		return CryptocurrencyString
	case Token:
		return TokenString
	case Contract:
		return ContractString
	case Stable:
		return StableString
	default:
		return UnsetRoleString
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
	case StableString:
		*r = Stable
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
			file.UnsetCurrency = append(file.UnsetCurrency, b.Items[i])
		case Fiat:
			file.FiatCurrency = append(file.FiatCurrency, b.Items[i])
		case Cryptocurrency:
			file.Cryptocurrency = append(file.Cryptocurrency, b.Items[i])
		case Token:
			file.Token = append(file.Token, b.Items[i])
		case Contract:
			file.Contracts = append(file.Contracts, b.Items[i])
		case Stable:
			file.Stable = append(file.Stable, b.Items[i])
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
		return fmt.Errorf("cannot update currency %w for %s", errRoleUnset, symbol)
	}
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol != symbol || (b.Items[i].Role != Unset && b.Items[i].Role != r) {
			continue
		}

		b.Items[i].FullName = fullName
		b.Items[i].Role = r
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

// Register registers a currency from a string and returns a currency code, this
// can optionally include a role when it is known.
func (b *BaseCodes) Register(c string, newRole Role) Code {
	if c == "" {
		return EMPTY
	}

	var format bool
	// Digits fool upper and lower casing. So find first letter and check case.
	for x := range c {
		if !unicode.IsDigit(rune(c[x])) {
			format = unicode.IsUpper(rune(c[x]))
			break
		}
	}

	// Force upper string storage and matching
	c = strings.ToUpper(c)

	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol != c {
			continue
		}

		if newRole != Unset {
			if b.Items[i].Role == Unset {
				b.Items[i].Role = newRole
			} else if b.Items[i].Role != newRole {
				// This will duplicate item with same name but different role.
				// TODO: This will need a specific update to NewCode to add in
				// a specific param to find the exact name and role.
				continue
			}
		}

		return Code{Item: b.Items[i], UpperCase: format}
	}
	newItem := &Item{Symbol: c, Lower: strings.ToLower(c), Role: newRole}
	b.Items = append(b.Items, newItem)
	return Code{Item: newItem, UpperCase: format}
}

// LoadItem sets item data
func (b *BaseCodes) LoadItem(item *Item) error {
	if item == nil {
		return errItemIsNil
	}

	if *item == (Item{}) {
		return errItemIsEmpty
	}

	item.Symbol = strings.ToUpper(item.Symbol)
	item.Lower = strings.ToLower(item.Symbol)

	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol != item.Symbol ||
			(b.Items[i].Role != Unset && item.Role != Unset && b.Items[i].Role != item.Role) {
			continue
		}
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
	return fmt.Sprintf("ID: %d Fullname: %s Symbol: %s Role: %s Chain: %s",
		i.ID,
		i.FullName,
		i.Symbol,
		i.Role,
		i.AssocChain)
}

// String converts the code to string
func (c Code) String() string {
	if c.Item == nil {
		return ""
	}
	if c.UpperCase {
		return c.Item.Symbol
	}
	return c.Item.Lower
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
	return c.Item == nil || c.Item.Symbol == ""
}

// Equal returns if the code supplied is the same as the corresponding code
func (c Code) Equal(check Code) bool {
	return c.Item == check.Item
}

// IsFiatCurrency checks if the currency passed is an enabled fiat currency
func (c Code) IsFiatCurrency() bool {
	return c.Item != nil && c.Item.Role == Fiat
}

// IsCryptocurrency checks if the currency passed is an enabled CRYPTO currency.
// NOTE: All unset currencies will default to cryptocurrencies and stable coins
// are cryptocurrencies as well.
func (c Code) IsCryptocurrency() bool {
	return c.Item != nil && c.Item.Role&(Cryptocurrency|Stable) == c.Item.Role
}

// IsStableCurrency checks if the currency is a stable currency.
func (c Code) IsStableCurrency() bool {
	return c.Item != nil && c.Item.Role == Stable
}
