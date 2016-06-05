package main

type IBotExchange interface {
	Setup(exch Exchanges)
	Start()
	SetDefaults()
	GetName() string
	IsEnabled() bool
   	GetTickerPrice(currency string) TickerPrice
}

