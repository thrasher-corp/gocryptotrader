// Package v19 renames the GateIO exchange to Gate, the name Gate.io now trades under.
package v19

import (
	"context"
	"encoding/json" //nolint:depguard // Config versions must retain stable standard-library JSON behaviour
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/buger/jsonparser"
)

var errExchangeNameConflict = errors.New("exchange is configured under both its old and new names")

const (
	oldName = "GateIO"
	newName = "Gate"
	// exchangeAddress describes a portfolio address holding an exchange's balance, which the portfolio names after the
	// exchange
	exchangeAddress = "Exchange"
)

// Version implements ConfigVersion to rename the GateIO exchange
type Version struct{}

// UpgradeConfig renames GateIO to Gate: the exchange, the portfolio's balances for it and the exchange lists of
// portfolio addresses and bank accounts, each name matched as the engine matches it. A config holding both names is
// refused, as one of the two exchanges would have to go
func (*Version) UpgradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return renameExchange(config, oldName, newName)
}

// DowngradeConfig renames Gate back to GateIO wherever UpgradeConfig renames GateIO
func (*Version) DowngradeConfig(_ context.Context, config []byte) ([]byte, error) {
	return renameExchange(config, newName, oldName)
}

func renameExchange(config []byte, from, to string) ([]byte, error) {
	exchanges, err := getArray(config, "exchanges")
	if err != nil {
		return config, err
	}
	var renames []int
	var held bool
	for i := range exchanges {
		var exchange struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(exchanges[i], &exchange); err != nil {
			return config, err
		}
		switch {
		case strings.EqualFold(exchange.Name, to):
			held = true
		case strings.EqualFold(exchange.Name, from):
			renames = append(renames, i)
		}
	}
	if held && len(renames) != 0 {
		return config, fmt.Errorf("%w: remove either %s or %s", errExchangeNameConflict, from, to)
	}
	for _, i := range renames {
		if config, err = setString(config, to, "exchanges", index(i), "name"); err != nil {
			return config, err
		}
	}
	renameBankAccount := func(account []byte) ([]byte, error) {
		return renameListed(account, "supportedExchanges", from, to)
	}
	for i := range exchanges {
		if config, err = editArray(config, renameBankAccount, "exchanges", index(i), "bankAccounts"); err != nil {
			return config, err
		}
	}
	if config, err = editArray(config, renameBankAccount, "bankAccounts"); err != nil {
		return config, err
	}
	return renamePortfolioAddresses(config, from, to)
}

// renamePortfolioAddresses renames the portfolio's balances for the exchange, which the portfolio matches by the
// exchange's exact name, and the exchange lists of its addresses. A balance of the same currency already held under
// the new name replaces the old one, as the portfolio keeps one balance for each exchange and currency
func renamePortfolioAddresses(config []byte, from, to string) ([]byte, error) {
	addresses, err := getArray(config, "portfolioAddresses", "addresses")
	if err != nil || len(addresses) == 0 {
		return config, err
	}
	held := make(map[string]bool)
	for _, address := range addresses {
		if exchange, coin, ok := exchangeBalance(address); ok && exchange == to {
			held[strings.ToUpper(coin)] = true
		}
	}
	renamed := make([]json.RawMessage, 0, len(addresses))
	var changed bool
	for _, address := range addresses {
		original := address
		if exchange, coin, ok := exchangeBalance(address); ok && exchange == from {
			if held[strings.ToUpper(coin)] {
				changed = true
				continue
			}
			if address, err = setString(address, to, "Address"); err != nil {
				return config, err
			}
		}
		if address, err = renameListed(address, "SupportedExchanges", from, to); err != nil {
			return config, err
		}
		changed = changed || string(address) != string(original)
		renamed = append(renamed, address)
	}
	if !changed {
		return config, nil
	}
	v, err := json.Marshal(renamed)
	if err != nil {
		return config, err
	}
	return set(config, v, "portfolioAddresses", "addresses")
}

// exchangeBalance returns the exchange and currency of a portfolio address holding an exchange's balance
func exchangeBalance(address []byte) (exchange, coin string, ok bool) {
	if description, err := jsonparser.GetString(address, resolve(address, "Description")...); err != nil || description != exchangeAddress {
		return "", "", false
	}
	exchange, err := jsonparser.GetString(address, resolve(address, "Address")...)
	if err != nil {
		return "", "", false
	}
	coin, _ = jsonparser.GetString(address, resolve(address, "CoinType")...)
	return exchange, coin, true
}

// renameListed renames the exchange in the comma separated exchange list held under key, matching each name in any
// case as bank accounts and portfolio addresses do, and keeping the spacing around it. A missing or non-string list
// names no exchange
func renameListed(element []byte, key, from, to string) ([]byte, error) {
	value, valueType, _, err := jsonparser.Get(element, resolve(element, key)...)
	if errors.Is(err, jsonparser.KeyPathNotFoundError) || err == nil && valueType != jsonparser.String {
		return element, nil
	}
	if err != nil {
		return element, err
	}
	list, err := jsonparser.ParseString(value)
	if err != nil {
		return element, err
	}
	names := strings.Split(list, ",")
	var changed bool
	for i, name := range names {
		if trimmed := strings.TrimSpace(name); strings.EqualFold(trimmed, from) {
			names[i] = strings.Replace(name, trimmed, to, 1)
			changed = true
		}
	}
	if !changed {
		return element, nil
	}
	return setString(element, strings.Join(names, ","), key)
}

// editArray applies edit to each element of the array at path, if there is one
func editArray(config []byte, edit func([]byte) ([]byte, error), path ...string) ([]byte, error) {
	elements, err := getArray(config, path...)
	if err != nil {
		return config, err
	}
	for i := range elements {
		element, err := edit(elements[i])
		if err != nil {
			return config, err
		}
		if string(element) == string(elements[i]) {
			continue
		}
		if config, err = set(config, element, append(path, index(i))...); err != nil {
			return config, err
		}
	}
	return config, nil
}

// getArray returns the elements of the array at path, or none when there is no array there
func getArray(config []byte, path ...string) ([]json.RawMessage, error) {
	value, valueType, _, err := jsonparser.Get(config, resolve(config, path...)...)
	switch {
	case errors.Is(err, jsonparser.KeyPathNotFoundError):
		return nil, nil
	case err != nil:
		return nil, err
	case valueType != jsonparser.Array:
		return nil, nil
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(value, &elements); err != nil {
		return nil, err
	}
	return elements, nil
}

func setString(data []byte, value string, path ...string) ([]byte, error) {
	v, err := json.Marshal(value)
	if err != nil {
		return data, err
	}
	return set(data, v, path...)
}

// set sets the value at path, keeping each key's spelling where it is already present
func set(data, value []byte, path ...string) ([]byte, error) {
	return jsonparser.Set(data, value, resolve(data, path...)...)
}

// resolve spells each key of path as data spells it, matching it in any case as the config loader does. Where several
// keys match, the last is taken, as its value is the one the loader keeps
func resolve(data []byte, path ...string) []string {
	resolved := slices.Clone(path)
	current := data
	for i, key := range path {
		if !strings.HasPrefix(key, "[") {
			_ = jsonparser.ObjectEach(current, func(found, _ []byte, _ jsonparser.ValueType, _ int) error {
				if strings.EqualFold(string(found), key) {
					resolved[i] = string(found)
				}
				return nil
			})
		}
		value, _, _, err := jsonparser.Get(current, resolved[i])
		if err != nil {
			break
		}
		current = value
	}
	return resolved
}

func index(i int) string {
	return "[" + strconv.Itoa(i) + "]"
}
