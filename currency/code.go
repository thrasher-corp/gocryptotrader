package currency

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/thrasher-corp/gocryptotrader/encoding/json"
)

var (
	// ErrCurrencyCodeEmpty defines an error if the currency code is empty
	ErrCurrencyCodeEmpty = errors.New("currency code is empty")
	// ErrCurrencyNotFound returned when a currency is not found in a list
	ErrCurrencyNotFound = errors.New("currency code not found in list")
	// ErrCurrencyPairEmpty defines an error if the currency pair is empty
	ErrCurrencyPairEmpty = errors.New("currency pair is empty")
	// ErrCurrencyNotSupported defines an error if the currency pair is not supported
	ErrCurrencyNotSupported = errors.New("currency not supported")
	// ErrCurrencyPairsEmpty returns when a currency.Pairs has len == 0
	ErrCurrencyPairsEmpty = errors.New("currency pairs is empty")
	// EMPTYCODE is an empty currency code
	EMPTYCODE = Code{}
	// EMPTYPAIR is an empty currency pair
	EMPTYPAIR = Pair{}

	errItemIsNil   = errors.New("item is nil")
	errItemIsEmpty = errors.New("item is empty")
	errRoleUnset   = errors.New("role unset")
)

// String implements the stringer interface and returns a string representation
// of the underlying role.
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
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for _, stored := range b.Items {
		for x := range stored {
			switch stored[x].Role {
			case Unset:
				file.UnsetCurrency = append(file.UnsetCurrency, stored[x])
			case Fiat:
				file.FiatCurrency = append(file.FiatCurrency, stored[x])
			case Cryptocurrency:
				file.Cryptocurrency = append(file.Cryptocurrency, stored[x])
			case Token:
				file.Token = append(file.Token, stored[x])
			case Contract:
				file.Contracts = append(file.Contracts, stored[x])
			case Stable:
				file.Stable = append(file.Stable, stored[x])
			default:
				return file, errors.New("role undefined")
			}
		}
	}
	file.LastMainUpdate = b.LastMainUpdate.Unix()
	return file, nil
}

// GetCurrencies gets the full currency list from the base code type available
// from the currency system
func (b *BaseCodes) GetCurrencies() Currencies {
	b.mtx.Lock()
	currencies := make(Currencies, len(b.Items))
	var target int
	for _, stored := range b.Items {
		if len(stored) == 0 {
			continue
		}
		currencies[target] = Code{Item: stored[0]}
		target++
	}
	b.mtx.Unlock()
	return currencies
}

// UpdateCurrency updates or registers a currency/contract
func (b *BaseCodes) UpdateCurrency(update *Item) error {
	if update == nil {
		return errItemIsNil
	}

	if update.Symbol == "" {
		return errSymbolEmpty
	}

	if update.Role == Unset {
		return fmt.Errorf("cannot update currency %w for %s", errRoleUnset, update.Symbol)
	}

	update.Symbol = strings.ToUpper(update.Symbol)
	update.Lower = strings.ToLower(update.Symbol)

	b.mtx.Lock()
	defer b.mtx.Unlock()

	stored, ok := b.Items[update.Symbol]
	if ok {
		for x := range stored {
			if stored[x].Role != Unset && stored[x].Role != update.Role {
				continue
			}

			stored[x].Role = update.Role // NOTE: Update role is checked above.

			if update.FullName != "" {
				stored[x].FullName = update.FullName
			}
			if update.AssocChain != "" {
				stored[x].AssocChain = update.AssocChain
			}
			if update.ID != 0 {
				stored[x].ID = update.ID
			}
			return nil
		}
	}
	b.Items[update.Symbol] = append(b.Items[update.Symbol], update)
	return nil
}

// Register registers a currency from a string and returns a currency code, this
// can optionally include a role when it is known.
func (b *BaseCodes) Register(c string, newRole Role) Code {
	if c == "" {
		return EMPTYCODE
	}

	isUpperCase := strings.ContainsFunc(c, func(r rune) bool { return unicode.IsLetter(r) && unicode.IsUpper(r) })

	// Force upper string storage and matching
	c = strings.ToUpper(c)

	b.mtx.Lock()
	defer b.mtx.Unlock()

	if b.Items == nil {
		b.Items = make(map[string][]*Item)
	}

	stored, ok := b.Items[c]
	if ok {
		for x := range stored {
			if newRole != Unset {
				if stored[x].Role != Unset && stored[x].Role != newRole {
					// This will duplicate item with same name but different
					// role if not matched in stored list.
					// TODO: This will need a specific update to NewCode() or
					// new function to find the exact name and role.
					continue
				}
				stored[x].Role = newRole
			}
			return Code{Item: stored[x], upperCase: isUpperCase}
		}
	}

	newItem := &Item{Symbol: c, Lower: strings.ToLower(c), Role: newRole}
	b.Items[c] = append(b.Items[c], newItem)
	return Code{Item: newItem, upperCase: isUpperCase}
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

	stored, ok := b.Items[item.Symbol]
	if ok {
		for x := range stored {
			if stored[x].Role == item.Role {
				return nil
			}

			if stored[x].Role == Unset && item.Role != Unset {
				stored[x].Role = item.Role
				return nil
			}
		}
	}
	b.Items[item.Symbol] = append(b.Items[item.Symbol], item)
	return nil
}

// NewCode returns a new currency registered code
func NewCode(c string) Code {
	return storage.ValidateCode(c)
}

// String conforms to the stringer interface
func (i *Item) String() string {
	return i.Symbol
}

// String converts the code to string
func (c Code) String() string {
	if c.Item == nil {
		return ""
	}
	if c.upperCase {
		return c.Item.Symbol
	}
	return c.Item.Lower
}

// Lower flags the Code to use LowerCase formatting, but does not change Symbol
// If Code cannot be lowercased then it will return Code unchanged
func (c Code) Lower() Code {
	if c.Item == nil {
		return c
	}
	c.upperCase = false
	return c
}

// Upper flags the Code to use UpperCase formatting, but does not change Symbol
// If Code cannot be uppercased then it will return Code unchanged
func (c Code) Upper() Code {
	if c.Item == nil {
		return c
	}
	c.upperCase = true
	return c
}

// UnmarshalJSON conforms type to the umarshaler interface
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

// Currency allows an item to revert to a code
func (i *Item) Currency() Code {
	if i == nil {
		return EMPTYCODE
	}
	return NewCode(i.Symbol)
}
