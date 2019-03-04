package currency

import (
	"fmt"
	"strings"
)

// NewCode returns a new currency registered code
func NewCode(c string) Code {
	return storage.NewCode(c)
}

// NewConversionFromString splits a string from a foreign exchange provider
func NewConversionFromString(p string) Conversion {
	return NewConversion(p[:3], p[3:])
}

// NewConversion assigns or finds a new conversion unit
func NewConversion(from, to string) Conversion {
	return Conversion{
		From: NewCode(from),
		To:   NewCode(to),
	}
}

// NewCurrenciesFromStrings returns a Currencies object from strings
func NewCurrenciesFromStrings(currencies []string) Currencies {
	var list Currencies
	for _, c := range currencies {
		if c == "" {
			continue
		}
		list = append(list, NewCode(c))
	}
	return list
}

// NewPairsFromStrings takes in currency pair strings and returns a currency
// pair list
func NewPairsFromStrings(pairs []string) Pairs {
	var ps Pairs
	for _, p := range pairs {
		if p == "" {
			continue
		}

		ps = append(ps, NewPairFromString(p))
	}
	return ps
}

// NewPairDelimiter splits the desired currency string at delimeter, the returns
// a Pair struct
func NewPairDelimiter(currencyPair, delimiter string) Pair {
	result := strings.Split(currencyPair, delimiter)
	return Pair{
		Delimiter: delimiter,
		Base:      NewCode(result[0]),
		Quote:     NewCode(result[1]),
	}
}

// NewPairFromStrings returns a CurrencyPair without a delimiter
func NewPairFromStrings(baseCurrency, quoteCurrency string) Pair {
	return Pair{
		Base:  NewCode(baseCurrency),
		Quote: NewCode(quoteCurrency),
	}
}

// NewPair returns a currency pair from currency codes
func NewPair(baseCurrency, quoteCurrency Code) Pair {
	return Pair{
		Base:  baseCurrency,
		Quote: quoteCurrency,
	}
}

// NewPairWithDelimiter returns a CurrencyPair with a delimiter
func NewPairWithDelimiter(base, quote, delimiter string) Pair {
	return Pair{
		Base:      NewCode(base),
		Quote:     NewCode(quote),
		Delimiter: delimiter,
	}
}

// NewPairFromIndex returns a CurrencyPair via a currency string and specific
// index
func NewPairFromIndex(currencyPair, index string) (Pair, error) {
	i := strings.Index(currencyPair, index)
	if i == -1 {
		return Pair{},
			fmt.Errorf("index %s not found in currency pair string", index)
	}
	if i == 0 {
		return NewPairFromStrings(currencyPair[0:len(index)],
				currencyPair[len(index):]),
			nil
	}
	return NewPairFromStrings(currencyPair[0:i], currencyPair[i:]), nil
}

// NewPairFromString converts currency string into a new CurrencyPair
// with or without delimeter
func NewPairFromString(currencyPair string) Pair {
	delimiters := []string{"_", "-"}
	var delimiter string
	for _, x := range delimiters {
		if strings.Contains(currencyPair, x) {
			delimiter = x
			return NewPairDelimiter(currencyPair, delimiter)
		}
	}
	return NewPairFromStrings(currencyPair[0:3], currencyPair[3:])
}

// NewConversionFromCode returns a conversion rate object that allows for
// obtaining efficient rate values when needed
func NewConversionFromCode(from, to Code) (Conversion, error) {
	return storage.NewConversion(from, to)
}

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

// GetDefaultBaseCurrency returns storage defauly base currency
func GetDefaultBaseCurrency() Code {
	return storage.GetDefaultBaseCurrency()
}

// GetCryptoCurrencies returns the storage enabled cryptocurrencies
func GetCryptoCurrencies() Currencies {
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

// RunStorageUpdater  runs a new foreign exchange updater instance
func RunStorageUpdater(o BotOverrides, m MainConfiguration, filepath string, v bool) error {
	return storage.RunUpdater(o, m, filepath, v)
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
		if delimiter != "" {
			p = NewPairDelimiter(pairs[x], delimiter)
		} else {
			if index != "" {
				var err error
				p, err = NewPairFromIndex(pairs[x], index)
				if err != nil {
					return Pairs{}, err
				}
			} else {
				p = NewPairFromStrings(pairs[x][0:3], pairs[x][3:])
			}
		}
		result = append(result, p)
	}
	return result, nil
}
