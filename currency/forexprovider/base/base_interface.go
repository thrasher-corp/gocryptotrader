package base

import (
	"errors"
	"log"
)

// IFXProviders contains an array of foreign exchange interfaces
type IFXProviders []IFXProvider

// IFXProvider enforces standard functions for all foreign exchange providers
// supported in GoCryptoTrader
type IFXProvider interface {
	Setup(config Settings)
	GetRates(baseCurrency, symbols string) (map[string]float64, error)
	GetName() string
	IsEnabled() bool
	IsPrimaryProvider() bool
}

// GetCurrencyData returns currency data from enabled FX providers
func (fxp IFXProviders) GetCurrencyData(baseCurrency, symbols string) (map[string]float64, error) {
	for x := range fxp {
		if fxp[x].IsPrimaryProvider() && fxp[x].IsEnabled() {
			rates, err := fxp[x].GetRates(baseCurrency, symbols)
			if err != nil {
				log.Println(err)
				for y := range fxp {
					if !fxp[y].IsPrimaryProvider() && fxp[x].IsEnabled() {
						rates, err = fxp[y].GetRates(baseCurrency, symbols)
						if err != nil {
							log.Println(err)
							continue
						}
						return rates, nil
					}
				}
				return nil, errors.New("ForexProvider error GetCurrencyData() failed to acquire data")
			}
			return rates, nil
		}
	}
	return nil, errors.New("ForexProvider error GetCurrencyData() no providers enabled")
}
