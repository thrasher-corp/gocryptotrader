// Package forexprovider utilises foreign exchange API services to manage
// relational FIAT currencies
package forexprovider

import (
	"errors"

	"github.com/thrasher-/gocryptotrader/currency/forexprovider/base"
	currencyconverter "github.com/thrasher-/gocryptotrader/currency/forexprovider/currencyconverterapi"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/currencylayer"
	exchangerates "github.com/thrasher-/gocryptotrader/currency/forexprovider/exchangeratesapi.io"
	fixer "github.com/thrasher-/gocryptotrader/currency/forexprovider/fixer.io"
	"github.com/thrasher-/gocryptotrader/currency/forexprovider/openexchangerates"
)

// ForexProviders is a foreign exchange handler type
type ForexProviders struct {
	base.FXHandler
}

// GetAvailableForexProviders returns a list of supported forex providers
func GetAvailableForexProviders() []string {
	return []string{"CurrencyConverter", "CurrencyLayer", "ExchangeRates", "Fixer", "OpenExchangeRates"}
}

// NewDefaultFXProvider returns the default forex provider (currencyconverterAPI)
func NewDefaultFXProvider() *ForexProviders {
	handler := new(ForexProviders)
	provider := new(exchangerates.ExchangeRates)
	provider.Setup(base.Settings{
		PrimaryProvider: true,
		Enabled:         true,
		Name:            "ExchangeRates",
	})

	currencies, _ := provider.GetSupportedCurrencies()
	providerBase := base.Provider{
		Provider:            provider,
		SupportedCurrencies: currencies,
	}

	handler.FXHandler = base.FXHandler{
		Primary: providerBase,
	}

	return handler
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
		switch {
		case fxProviders[i].Name == "CurrencyConverter" && fxProviders[i].Enabled:
			provider := new(currencyconverter.CurrencyConverter)
			provider.Setup(fxProviders[i])
			handler.SetProvider(provider)

		case fxProviders[i].Name == "CurrencyLayer" && fxProviders[i].Enabled:
			provider := new(currencylayer.CurrencyLayer)
			provider.Setup(fxProviders[i])
			handler.SetProvider(provider)

		case fxProviders[i].Name == "ExchangeRates" && fxProviders[i].Enabled:
			provider := new(exchangerates.ExchangeRates)
			provider.Setup(fxProviders[i])
			handler.SetProvider(provider)

		case fxProviders[i].Name == "Fixer" && fxProviders[i].Enabled:
			provider := new(fixer.Fixer)
			provider.Setup(fxProviders[i])
			handler.SetProvider(provider)

		case fxProviders[i].Name == "OpenExchangeRates" && fxProviders[i].Enabled:
			provider := new(openexchangerates.OXR)
			provider.Setup(fxProviders[i])
			handler.SetProvider(provider)
		}
	}

	if handler.Primary.Provider == nil {
		return nil, errors.New("no foreign exchange providers enabled")
	}

	return handler, nil
}
