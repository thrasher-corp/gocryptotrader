package pair

import "strings"

type CurrencyItem string

func (c CurrencyItem) Lower() CurrencyItem {
	return CurrencyItem(strings.ToLower(string(c)))
}

func (c CurrencyItem) Upper() CurrencyItem {
	return CurrencyItem(strings.ToUpper(string(c)))
}

func (c CurrencyItem) String() string {
	return string(c)
}

type CurrencyPair struct {
	Delimiter      string       `json:"delimiter"`
	FirstCurrency  CurrencyItem `json:"first_currency"`
	SecondCurrency CurrencyItem `json:"second_currency"`
}

func (c CurrencyPair) GetFirstCurrency() CurrencyItem {
	return c.FirstCurrency
}

func (c CurrencyPair) GetSecondCurrency() CurrencyItem {
	return c.SecondCurrency
}

func (c CurrencyPair) Pair() CurrencyItem {
	return c.FirstCurrency + CurrencyItem(c.Delimiter) + c.SecondCurrency
}

func NewCurrencyPairDelimiter(currency, delimiter string) CurrencyPair {
	result := strings.Split(currency, delimiter)
	return CurrencyPair{
		Delimiter:      delimiter,
		FirstCurrency:  CurrencyItem(result[0]),
		SecondCurrency: CurrencyItem(result[1]),
	}
}

func NewCurrencyPair(firstCurrency, secondCurrency string) CurrencyPair {
	return CurrencyPair{
		FirstCurrency:  CurrencyItem(firstCurrency),
		SecondCurrency: CurrencyItem(secondCurrency),
	}
}

func NewCurrencyPairFromString(currency string) CurrencyPair {
	delmiters := []string{"_", "-"}
	var delimiter string
	for _, x := range delmiters {
		if strings.Contains(currency, x) {
			delimiter = x
			return NewCurrencyPairDelimiter(currency, delimiter)
		}
	}
	return NewCurrencyPair(currency[0:3], currency[3:])
}
