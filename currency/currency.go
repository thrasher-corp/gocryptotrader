package currency

import (
	"strings"
)

func init() {
	system.SetDefaults()
}

// NewCurrencyCode returns a new currency registered code
func NewCurrencyCode(c string) Code {
	return system.NewCurrencyCode(c)
}

// NewConversionFromString splits a string from a foreign exchange provider
func NewConversionFromString(p string) Conversion {
	return NewConversion(p[:3], p[3:])
}

// NewConversion assigns or finds a new conversion unit
func NewConversion(from, to string) Conversion {
	return Conversion{
		From: NewCurrencyCode(from),
		To:   NewCurrencyCode(to),
	}
}

// NewCurrenciesFromStrings returns a Currencies object from strings
func NewCurrenciesFromStrings(currencies []string) Currencies {
	var list Currencies
	for _, c := range currencies {
		if c == "" {
			continue
		}
		list = append(list, NewCurrencyCode(c))
	}
	return list
}

// NewPairsFromStrings takes in currency pair strings and returns a
// currency pair list
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
		Base:      NewCurrencyCode(result[0]),
		Quote:     NewCurrencyCode(result[1]),
	}
}

// NewPair returns a CurrencyPair without a delimiter
func NewPair(baseCurrency, quoteCurrency string) Pair {
	return Pair{
		Base:  NewCurrencyCode(baseCurrency),
		Quote: NewCurrencyCode(quoteCurrency),
	}
}

// NewPairFromCodes returns a currency pair from currency codes
func NewPairFromCodes(baseCurrency, quoteCurrency Code) Pair {
	return Pair{
		Base:  baseCurrency,
		Quote: quoteCurrency,
	}
}

// NewPairWithDelimiter returns a CurrencyPair with a delimiter
func NewPairWithDelimiter(base, quote, delimiter string) Pair {
	return Pair{
		Base:      NewCurrencyCode(base),
		Quote:     NewCurrencyCode(quote),
		Delimiter: delimiter,
	}
}

// NewPairFromIndex returns a CurrencyPair via a currency string and specific
// index
func NewPairFromIndex(currencyPair, index string) Pair {
	i := strings.Index(currencyPair, index)
	if i == 0 {
		return NewPair(currencyPair[0:len(index)], currencyPair[len(index):])
	}
	return NewPair(currencyPair[0:i], currencyPair[i:])
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
	return NewPair(currencyPair[0:3], currencyPair[3:])
}

// NewConversionFromCode returns a conversion rate abject that allows for
// obtaining efficient rate values when needed
func NewConversionFromCode(from, to Code) (Conversion, error) {
	return system.NewConversion(from, to)
}

// GetDefaultExchangeRates returns the currency exchange rates based off the
// default fiat values
func GetDefaultExchangeRates() (Conversions, error) {
	return system.GetDefaultForeignExchangeRates()
}

// GetSystemExchangeRates returns the full fiat currency exchange rates base off
// configuration parameters supplied to the system
func GetSystemExchangeRates() (Conversions, error) {
	return system.GetSystemExchangeRates()
}

// UpdateBaseCurrency updates system base currency
func UpdateBaseCurrency(c Code) error {
	return system.UpdateBaseCurrency(c)
}

// GetBaseCurrency returns the system base currency
func GetBaseCurrency() Code {
	return system.GetBaseCurrency()
}

// GetDefaultBaseCurrency returns system defauly base currency
func GetDefaultBaseCurrency() Code {
	return system.GetDefaultBaseCurrency()
}

// GetSystemCryptoCurrencies returns the system enabled cryptocurrencies
func GetSystemCryptoCurrencies() Currencies {
	return system.GetCryptocurrencies()
}

// GetDefaultCryptocurrencies returns a list of default cryptocurrencies
func GetDefaultCryptocurrencies() Currencies {
	return system.GetDefaultCryptocurrencies()
}

// GetSystemFiatCurrencies returns the system enabled fiat currencies
func GetSystemFiatCurrencies() Currencies {
	return system.GetFiatCurrencies()
}

// GetDefaultFiatCurrencies returns a list of default fiat currencies
func GetDefaultFiatCurrencies() Currencies {
	return system.GetDefaultFiatCurrencies()
}

// UpdateCurrencies updates the local cryptocurrency or fiat currency store
func UpdateCurrencies(c Currencies, cryptos bool) {
	if cryptos {
		system.UpdateEnabledCryptoCurrencies(c)
		return
	}
	system.UpdateEnabledFiatCurrencies(c)
}

// ConvertCurrency converts an amount from one currency to another
func ConvertCurrency(amount float64, from, to Code) (float64, error) {
	return system.ConvertCurrency(amount, from, to)
}

// SeedForiegnExchangeData seeds FX data with the currencies supplied
func SeedForiegnExchangeData(c Currencies) error {
	return system.SeedForeignExchangeRatesByCurrencies(c)
}

// GetTotalMarketCryptocurrencies returns the full market cryptocurrencies
func GetTotalMarketCryptocurrencies() ([]Code, error) {
	return system.GetTotalMarketCryptocurrencies()
}

// // GetTotalMarketExchanges returns the full market exchange participation data
// func GetTotalMarketExchanges() []Data {
// 	return system.GetTotalMarketExchanges()
// }

// RunUpdaterSystem runs sets up and runs a new foreign exchange updater
// instance
func RunUpdaterSystem(o BotOverrides, m MainConfiguration, filepath string, v bool) error {
	return system.RunUpdater(o, m, filepath, v)
}

// CopyPairFormat copies the pair format from a list of pairs once matched
// NOTE: Unused in codebase
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
	return Pair{Base: NewCurrencyCode(""), Quote: NewCurrencyCode("")}
}

// FormatPairs formats a string array to a list of currency pairs with the
// supplied currency pair format
// NOTE: Unused in codebase
func FormatPairs(pairs []string, delimiter, index string) Pairs {
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
				p = NewPairFromIndex(pairs[x], index)
			} else {
				p = NewPair(pairs[x][0:3], pairs[x][3:])
			}
		}
		result = append(result, p)
	}
	return result
}
