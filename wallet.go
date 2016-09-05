package main

//ExchangeAccountInfo : Generic type to hold each exchange's holdings in all enabled currencies
type ExchangeAccountInfo struct {
	Currencies []ExchangeAccountCurrencyInfo
}

//ExchangeAccountCurrencyInfo : Sub type to store currency name and value
type ExchangeAccountCurrencyInfo struct {
	CurrencyName string
	TotalValue   int
	Hold         int
}
