package currency

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/thrasher-corp/gocryptotrader/common"
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
	for _, i := range b.Items {
		switch i.Role {
		case Unset:
			file.UnsetCurrency = append(file.UnsetCurrency, *i)
		case Fiat:
			file.FiatCurrency = append(file.FiatCurrency, *i)
		case Cryptocurrency:
			file.Cryptocurrency = append(file.Cryptocurrency, *i)
		case Token:
			file.Token = append(file.Token, *i)
		case Contract:
			file.Contracts = append(file.Contracts, *i)
		default:
			return file, errors.New("role undefined")
		}
	}

	file.LastMainUpdate = b.LastMainUpdate
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

// UpdateCryptocurrency updates or registers a cryptocurrency
func (b *BaseCodes) UpdateCryptocurrency(fullName, symbol string, id int) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol != symbol {
			continue
		}
		if b.Items[i].Role != Unset {
			if b.Items[i].Role != Cryptocurrency {
				if b.Items[i].FullName != "" {
					if b.Items[i].FullName != fullName {
						// multiple symbols found, break this and add the
						// full context - this most likely won't occur for
						// fiat but could occur for contracts.
						break
					}
				}
				return fmt.Errorf("role already defined in cryptocurrency %s as [%s]",
					b.Items[i].Symbol,
					b.Items[i].Role)
			}
			b.Items[i].FullName = fullName
			b.Items[i].ID = id
			return nil
		}

		b.Items[i].Role = Cryptocurrency
		b.Items[i].FullName = fullName
		b.Items[i].ID = id
		return nil
	}

	b.Items = append(b.Items, &Item{
		FullName: fullName,
		Symbol:   symbol,
		ID:       id,
		Role:     Cryptocurrency,
	})
	return nil
}

// UpdateFiatCurrency updates or registers a fiat currency
func (b *BaseCodes) UpdateFiatCurrency(fullName, symbol string, id int) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol != symbol {
			continue
		}

		if b.Items[i].Role != Unset {
			if b.Items[i].Role != Fiat {
				return fmt.Errorf("role already defined in fiat currency %s as [%s]",
					b.Items[i].Symbol,
					b.Items[i].Role)
			}
			b.Items[i].FullName = fullName
			b.Items[i].ID = id
			return nil
		}

		b.Items[i].Role = Fiat
		b.Items[i].FullName = fullName
		b.Items[i].ID = id
		return nil
	}

	b.Items = append(b.Items, &Item{
		FullName: fullName,
		Symbol:   symbol,
		ID:       id,
		Role:     Fiat,
	})
	return nil
}

// UpdateToken updates or registers a token
func (b *BaseCodes) UpdateToken(fullName, symbol, assocBlockchain string, id int) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol != symbol {
			continue
		}

		if b.Items[i].Role != Unset {
			if b.Items[i].Role != Token {
				if b.Items[i].FullName != "" {
					if b.Items[i].FullName != fullName {
						// multiple symbols found, break this and add the
						// full context - this most likely won't occur for
						// fiat but could occur for contracts.
						break
					}
				}
				return fmt.Errorf("role already defined in token %s as [%s]",
					b.Items[i].Symbol,
					b.Items[i].Role)
			}
			b.Items[i].FullName = fullName
			b.Items[i].ID = id
			b.Items[i].AssocChain = assocBlockchain
			return nil
		}

		b.Items[i].Role = Token
		b.Items[i].FullName = fullName
		b.Items[i].ID = id
		b.Items[i].AssocChain = assocBlockchain
		return nil
	}

	b.Items = append(b.Items, &Item{
		FullName:   fullName,
		Symbol:     symbol,
		ID:         id,
		Role:       Token,
		AssocChain: assocBlockchain,
	})
	return nil
}

// UpdateContract updates or registers a contract
func (b *BaseCodes) UpdateContract(fullName, symbol, assocExchange string) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol != symbol {
			continue
		}

		if b.Items[i].Role != Unset {
			if b.Items[i].Role != Contract {
				return fmt.Errorf("role already defined in contract %s as [%s]",
					b.Items[i].Symbol,
					b.Items[i].Role)
			}
			b.Items[i].FullName = fullName
			if !common.StringDataContains(b.Items[i].AssocExchange, assocExchange) {
				b.Items[i].AssocExchange = append(b.Items[i].AssocExchange,
					assocExchange)
			}
			return nil
		}

		b.Items[i].Role = Contract
		b.Items[i].FullName = fullName
		if !common.StringDataContains(b.Items[i].AssocExchange, assocExchange) {
			b.Items[i].AssocExchange = append(b.Items[i].AssocExchange,
				assocExchange)
		}
		return nil
	}

	b.Items = append(b.Items, &Item{
		FullName:      fullName,
		Symbol:        symbol,
		Role:          Contract,
		AssocExchange: []string{assocExchange},
	})
	return nil
}

// Register registers a currency from a string and returns a currency code
func (b *BaseCodes) Register(c string) Code {
	NewUpperCode := c
	lower := true
	for _, r := range c {
		if !unicode.IsLower(r) {
			lower = false
		}
	}
	if lower {
		NewUpperCode = strings.ToUpper(c)
	}
	format := strings.Contains(c, NewUpperCode)

	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol == NewUpperCode {
			return Code{
				Item:      b.Items[i],
				UpperCase: format,
			}
		}
	}

	newItem := Item{Symbol: NewUpperCode}
	newCode := Code{
		Item:      &newItem,
		UpperCase: format,
	}

	b.Items = append(b.Items, newCode.Item)
	return newCode
}

// RegisterFiat registers a fiat currency from a string and returns a currency
// code
func (b *BaseCodes) RegisterFiat(c string) (Code, error) {
	c = strings.ToUpper(c)

	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol == c {
			if b.Items[i].Role != Unset {
				if b.Items[i].Role != Fiat {
					return Code{}, fmt.Errorf("register fiat error role already defined in fiat %s as [%s]",
						b.Items[i].Symbol,
						b.Items[i].Role)
				}
				return Code{Item: b.Items[i], UpperCase: true}, nil
			}
			b.Items[i].Role = Fiat
			return Code{Item: b.Items[i], UpperCase: true}, nil
		}
	}

	item := &Item{Symbol: c, Role: Fiat}
	b.Items = append(b.Items, item)

	return Code{Item: item, UpperCase: true}, nil
}

// LoadItem sets item data
func (b *BaseCodes) LoadItem(item *Item) error {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	for i := range b.Items {
		if b.Items[i].Symbol == item.Symbol {
			if b.Items[i].Role == Unset {
				b.Items[i].AssocChain = item.AssocChain
				b.Items[i].AssocExchange = item.AssocExchange
				b.Items[i].ID = item.ID
				b.Items[i].Role = item.Role
				b.Items[i].FullName = item.FullName
				return nil
			}

			if b.Items[i].FullName != "" {
				if b.Items[i].FullName == item.FullName {
					b.Items[i].AssocChain = item.AssocChain
					b.Items[i].AssocExchange = item.AssocExchange
					b.Items[i].ID = item.ID
					b.Items[i].Role = item.Role
					return nil
				}
				break
			}

			if b.Items[i].ID == item.ID {
				b.Items[i].AssocChain = item.AssocChain
				b.Items[i].AssocExchange = item.AssocExchange
				b.Items[i].FullName = item.FullName
				b.Items[i].ID = item.ID
				b.Items[i].Role = item.Role
				return nil
			}

			return fmt.Errorf("currency %s not found in currencycode list",
				item.Symbol)
		}
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
		return c.Item.Symbol
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
