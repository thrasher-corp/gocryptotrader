// Package forexprovider utilises foreign exchange API services to manage
// relational FIAT currencies
package forexprovider

import (
	"log"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/currencylayer"
	fixer "github.com/thrasher-/gocryptotrader/currency/forexprovider/fixer.io"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/openexchangerates"
)

// ForexProviders is an array of foreign exchange interfaces
type ForexProviders struct {
	base.IFXProviders
}

// StartFXService starts the forex provider service and returns a pointer to it
func StartFXService(config []config.ForexProviderConfig) *ForexProviders {
	fxp := new(ForexProviders)
	for i := range config {
		if config[i].Name == "CurrencyLayer" && config[i].Enabled {
			currencyLayerP := new(currencylayer.CurrencyLayer)
			currencyLayerP.Setup(config[i])
			fxp.IFXProviders = append(fxp.IFXProviders, currencyLayerP)
		}
		if config[i].Name == "Fixer" && config[i].Enabled {
			fixerP := new(fixer.Fixer)
			fixerP.Setup(config[i])
			fxp.IFXProviders = append(fxp.IFXProviders, fixerP)
		}
		if config[i].Name == "OpenExchangeRates" && config[i].Enabled {
			OpenExchangeRatesP := new(openexchangerates.OXR)
			OpenExchangeRatesP.Setup(config[i])
			fxp.IFXProviders = append(fxp.IFXProviders, OpenExchangeRatesP)
		}
	}
	if len(fxp.IFXProviders) == 0 {
		log.Fatal("No foreign exchange providers enabled")
	}
	return fxp
}
