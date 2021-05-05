// Package forexprovider utilises foreign exchange API services to manage
// relational FIAT currencies
package forexprovider

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	currencyconverter "github.com/thrasher-corp/gocryptotrader/currency/forexprovider/currencyconverterapi"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/currencylayer"
	exchangeratehost "github.com/thrasher-corp/gocryptotrader/currency/forexprovider/exchangerate.host"
	exchangerates "github.com/thrasher-corp/gocryptotrader/currency/forexprovider/exchangeratesapi.io"
	fixer "github.com/thrasher-corp/gocryptotrader/currency/forexprovider/fixer.io"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/openexchangerates"
)

// GetSupportedForexProviders returns a list of supported forex providers
func GetSupportedForexProviders() []string {
	return []string{
		"CurrencyConverter",
		"CurrencyLayer",
		"ExchangeRates",
		"Fixer",
		"OpenExchangeRates",
		"ExchangeRateHost",
	}
}

// NewDefaultFXProvider returns the default forex provider (currencyconverterAPI)
func NewDefaultFXProvider() *ForexProviders {
	handler := new(ForexProviders)
	provider := new(exchangeratehost.ExchangeRateHost)
	err := provider.Setup(base.Settings{
		PrimaryProvider: true,
		Enabled:         true,
		Name:            "ExchangeRateHost",
	})
	if err != nil {
		panic(err)
	}

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
			err := provider.Setup(fxProviders[i])
			if err != nil {
				return nil, err
			}

			handler.SetProvider(provider)

		case fxProviders[i].Name == "CurrencyLayer" && fxProviders[i].Enabled:
			provider := new(currencylayer.CurrencyLayer)
			err := provider.Setup(fxProviders[i])
			if err != nil {
				return nil, err
			}

			handler.SetProvider(provider)

		case fxProviders[i].Name == "ExchangeRates" && fxProviders[i].Enabled:
			provider := new(exchangerates.ExchangeRates)
			err := provider.Setup(fxProviders[i])
			if err != nil {
				return nil, err
			}

			handler.SetProvider(provider)

		case fxProviders[i].Name == "Fixer" && fxProviders[i].Enabled:
			provider := new(fixer.Fixer)
			err := provider.Setup(fxProviders[i])
			if err != nil {
				return nil, err
			}

			handler.SetProvider(provider)

		case fxProviders[i].Name == "OpenExchangeRates" && fxProviders[i].Enabled:
			provider := new(openexchangerates.OXR)
			err := provider.Setup(fxProviders[i])
			if err != nil {
				return nil, err
			}

			handler.SetProvider(provider)
		}
	}

	if handler.Primary.Provider == nil {
		return nil, errors.New("no primary forex provider enabled")
	}

	return handler, nil
}

// ForexProviders is a foreign exchange handler type
type ForexProviders struct {
	base.FXHandler
}
