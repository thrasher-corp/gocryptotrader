package main

//IBotExchange : Enforces standard functions for all exchanges supported in gocryptotrader
type IBotExchange interface {
	Setup(exch Exchanges)
	Start()
	SetDefaults()
	GetName() string
	IsEnabled() bool
	GetTickerPrice(currency string) TickerPrice
	GetEnabledCurrencies() []string
	GetExchangeAccountInfo() ExchangeAccountInfo
}
