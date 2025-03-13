package currency

import (
	"errors"
	"fmt"
	"strings"
)

var errEmptyPairString = errors.New("empty pair string")

// GetDefaultExchangeRates returns the currency exchange rates based off the
// default fiat values
func GetDefaultExchangeRates() (Conversions, error) {
	return storage.GetDefaultForeignExchangeRates()
}

// GetExchangeRates returns the full fiat currency exchange rates base off
// configuration parameters supplied to the currency storage
func GetExchangeRates() (Conversions, error) {
	return storage.GetExchangeRates()
}

// UpdateBaseCurrency updates storage base currency
func UpdateBaseCurrency(c Code) error {
	return storage.UpdateBaseCurrency(c)
}

// GetBaseCurrency returns the storage base currency
func GetBaseCurrency() Code {
	return storage.GetBaseCurrency()
}

// GetDefaultBaseCurrency returns storage default base currency
func GetDefaultBaseCurrency() Code {
	return storage.GetDefaultBaseCurrency()
}

// GetCryptocurrencies returns the storage enabled cryptocurrencies
func GetCryptocurrencies() Currencies {
	return storage.GetCryptocurrencies()
}

// GetDefaultCryptocurrencies returns a list of default cryptocurrencies
func GetDefaultCryptocurrencies() Currencies {
	return storage.GetDefaultCryptocurrencies()
}

// GetFiatCurrencies returns the storage enabled fiat currencies
func GetFiatCurrencies() Currencies {
	return storage.GetFiatCurrencies()
}

// GetDefaultFiatCurrencies returns a list of default fiat currencies
func GetDefaultFiatCurrencies() Currencies {
	return storage.GetDefaultFiatCurrencies()
}

// UpdateCurrencies updates the local cryptocurrency or fiat currency store
func UpdateCurrencies(c Currencies, isCryptocurrency bool) {
	if isCryptocurrency {
		storage.UpdateEnabledCryptoCurrencies(c)
		return
	}
	storage.UpdateEnabledFiatCurrencies(c)
}

// ConvertFiat converts a fiat amount from one currency to another
func ConvertFiat(amount float64, from, to Code) (float64, error) {
	return storage.ConvertCurrency(amount, from, to)
}

// GetForeignExchangeRate returns the foreign exchange rate for a fiat pair.
func GetForeignExchangeRate(quotation Pair) (float64, error) {
	return storage.ConvertCurrency(1, quotation.Base, quotation.Quote)
}

// SeedForeignExchangeData seeds FX data with the currencies supplied
func SeedForeignExchangeData(c Currencies) error {
	return storage.SeedForeignExchangeRatesByCurrencies(c)
}

// GetTotalMarketCryptocurrencies returns the full market cryptocurrencies
func GetTotalMarketCryptocurrencies() ([]Code, error) {
	return storage.GetTotalMarketCryptocurrencies()
}

// RunStorageUpdater runs a new foreign exchange updater instance
func RunStorageUpdater(o BotOverrides, m *Config, filepath string) error {
	return storage.RunUpdater(o, m, filepath)
}

// ShutdownStorageUpdater cleanly shuts down and saves to currency.json
func ShutdownStorageUpdater() error {
	return storage.Shutdown()
}

// CopyPairFormat copies the pair format from a list of pairs once matched
func CopyPairFormat(p Pair, pairs []Pair, exact bool) Pair {
	for x := range pairs {
		if exact {
			if p.Equal(pairs[x]) {
				return pairs[x]
			}
			continue
		}
		if p.EqualIncludeReciprocal(pairs[x]) {
			return pairs[x]
		}
	}
	return EMPTYPAIR
}

// FormatPairs formats a string array to a list of currency pairs with the supplied currency pair format
func FormatPairs(pairs []string, delimiter string) (Pairs, error) {
	result := make(Pairs, len(pairs))
	for x := range pairs {
		if pairs[x] == "" {
			return nil, fmt.Errorf("%w in slice %v", errEmptyPairString, pairs)
		}
		var err error
		switch {
		case delimiter != "":
			result[x], err = NewPairDelimiter(pairs[x], delimiter)
		case len(pairs[x]) < 3:
			err = errNoDelimiter
		default:
			result[x], err = NewPairFromStrings(pairs[x][:3], pairs[x][3:])
		}
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// IsEnabled returns if the individual foreign exchange config setting is
// enabled
func (settings AllFXSettings) IsEnabled(name string) bool {
	for x := range settings {
		if !strings.EqualFold(settings[x].Name, name) {
			continue
		}
		return settings[x].Enabled
	}
	return false
}
