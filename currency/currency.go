package currency

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

// ConvertCurrency converts an amount from one currency to another
func ConvertCurrency(amount float64, from, to Code) (float64, error) {
	return storage.ConvertCurrency(amount, from, to)
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
func RunStorageUpdater(o BotOverrides, m *MainConfiguration, filepath string) error {
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
		}
		if p.EqualIncludeReciprocal(pairs[x]) {
			return pairs[x]
		}
	}
	return Pair{Base: NewCode(""), Quote: NewCode("")}
}

// FormatPairs formats a string array to a list of currency pairs with the
// supplied currency pair format
func FormatPairs(pairs []string, delimiter, index string) (Pairs, error) {
	var result Pairs
	for x := range pairs {
		if pairs[x] == "" {
			continue
		}
		var p Pair
		var err error
		if delimiter != "" {
			p, err = NewPairDelimiter(pairs[x], delimiter)
			if err != nil {
				return nil, err
			}
		} else {
			if index != "" {
				p, err = NewPairFromIndex(pairs[x], index)
				if err != nil {
					return Pairs{}, err
				}
			} else {
				p, err = NewPairFromStrings(pairs[x][0:3], pairs[x][3:])
				if err != nil {
					return Pairs{}, err
				}
			}
		}
		result = append(result, p)
	}
	return result, nil
}
