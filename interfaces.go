package main

import (
	"github.com/thrasher-/gocryptotrader/config"
)

//IBotExchange : Enforces standard functions for all exchanges supported in gocryptotrader
type IBotExchange interface {
	Setup(exch config.ExchangeConfig)
	Start()
	SetDefaults()
	GetName() string
	IsEnabled() bool
	GetTickerPrice(currency string) (TickerPrice, error)
	GetEnabledCurrencies() []string
	GetExchangeAccountInfo() (ExchangeAccountInfo, error)
}
