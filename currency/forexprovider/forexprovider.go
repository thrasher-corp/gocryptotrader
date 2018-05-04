// Package forexprovider utilises foreign exchange API services to manage
// relational FIAT currencies
package forexprovider

import (
	"log"

	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	currencyconverter "github.com/thrasher-/gocryptotrader/currency/forexprovider/currencyconverterapi"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/currencylayer"
	fixer "github.com/thrasher-/gocryptotrader/currency/forexprovider/fixer.io"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/openexchangerates"
)

// ForexProviders is an array of foreign exchange interfaces
type ForexProviders struct {
	base.IFXProviders
}

// GetAvailableForexProviders returns a list of supported forex providers
func GetAvailableForexProviders() []string {
	return []string{"CurrencyConverter", "CurrencyLayer", "Fixer", "OpenExchangeRates"}
}

// NewDefaultFXProvider returns the default forex provider (currencyconverterAPI)
func NewDefaultFXProvider() *ForexProviders {
	fxp := new(ForexProviders)
	currencyC := new(currencyconverter.CurrencyConverter)
	currencyC.PrimaryProvider = true
	currencyC.Enabled = true
	currencyC.Name = "CurrencyConverter"
	currencyC.APIKeyLvl = 0
	currencyC.Verbose = false
	fxp.IFXProviders = append(fxp.IFXProviders, currencyC)
	return fxp
}

// StartFXService starts the forex provider service and returns a pointer to it
func StartFXService(fxProviders []base.Settings) *ForexProviders {
	fxp := new(ForexProviders)
	for i := range fxProviders {
		if fxProviders[i].Name == "CurrencyConverter" && fxProviders[i].Enabled {
			currencyC := new(currencyconverter.CurrencyConverter)
			currencyC.Setup(fxProviders[i])
			fxp.IFXProviders = append(fxp.IFXProviders, currencyC)
		}
		if fxProviders[i].Name == "CurrencyLayer" && fxProviders[i].Enabled {
			currencyLayerP := new(currencylayer.CurrencyLayer)
			currencyLayerP.Setup(fxProviders[i])
			fxp.IFXProviders = append(fxp.IFXProviders, currencyLayerP)
		}
		if fxProviders[i].Name == "Fixer" && fxProviders[i].Enabled {
			fixerP := new(fixer.Fixer)
			fixerP.Setup(fxProviders[i])
			fxp.IFXProviders = append(fxp.IFXProviders, fixerP)
		}
		if fxProviders[i].Name == "OpenExchangeRates" && fxProviders[i].Enabled {
			OpenExchangeRatesP := new(openexchangerates.OXR)
			OpenExchangeRatesP.Setup(fxProviders[i])
			fxp.IFXProviders = append(fxp.IFXProviders, OpenExchangeRatesP)
		}
	}
	if len(fxp.IFXProviders) == 0 {
		log.Fatal("No foreign exchange providers enabled")
	}
	return fxp
}
