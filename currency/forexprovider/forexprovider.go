// Package forexprovider utilises foreign exchange API services to manage
// relational FIAT currencies
package forexprovider

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	currencyconverter "github.com/thrasher-corp/gocryptotrader/currency/forexprovider/currencyconverterapi"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/currencylayer"
	exchangerates "github.com/thrasher-corp/gocryptotrader/currency/forexprovider/exchangeratesapi.io"
	fixer "github.com/thrasher-corp/gocryptotrader/currency/forexprovider/fixer.io"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/openexchangerates"
)

var (
	errUnhandledForeignExchangeProvider = errors.New("unhandled foreign exchange provider")
	errNoPrimaryForexProviderEnabled    = errors.New("no primary forex provider enabled")
)

// ForexProviders is a foreign exchange handler type
type ForexProviders struct {
	base.FXHandler
}

// GetSupportedForexProviders returns a list of supported forex providers
func GetSupportedForexProviders() []string {
	return []string{
		"CurrencyConverter",
		"CurrencyLayer",
		"ExchangeRates",
		"Fixer",
		"OpenExchangeRates",
	}
}

// SetProvider sets provider to the FX handler
func (f *ForexProviders) SetProvider(b base.IFXProvider) error {
	currencies, err := b.GetSupportedCurrencies()
	if err != nil {
		return err
	}

	providerBase := base.Provider{
		Provider:            b,
		SupportedCurrencies: currencies,
	}

	if b.IsPrimaryProvider() {
		f.FXHandler = base.FXHandler{
			Primary: providerBase,
		}
		return nil
	}

	f.FXHandler.Support = append(f.FXHandler.Support, providerBase)
	return nil
}

// StartFXService starts the forex provider service and returns a pointer to it
func StartFXService(fxProviders []base.Settings) (*ForexProviders, error) {
	handler := new(ForexProviders)
	for i := range fxProviders {
		var provider base.IFXProvider
		switch fxProviders[i].Name {
		case "CurrencyConverter":
			provider = new(currencyconverter.CurrencyConverter)
		case "CurrencyLayer":
			provider = new(currencylayer.CurrencyLayer)
		case "ExchangeRates":
			provider = new(exchangerates.ExchangeRates)
		case "Fixer":
			provider = new(fixer.Fixer)
		case "OpenExchangeRates":
			provider = new(openexchangerates.OXR)
		default:
			return nil, fmt.Errorf("%s %w", fxProviders[i].Name,
				errUnhandledForeignExchangeProvider)
		}
		err := provider.Setup(fxProviders[i])
		if err != nil {
			return nil, err
		}
		err = handler.SetProvider(provider)
		if err != nil {
			return nil, err
		}
	}

	if handler.Primary.Provider == nil {
		return nil, errNoPrimaryForexProviderEnabled
	}

	return handler, nil
}
