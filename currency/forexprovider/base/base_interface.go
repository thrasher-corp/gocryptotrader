package base

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// IFXProvider enforces standard functions for all foreign exchange providers
// supported in GoCryptoTrader
type IFXProvider interface {
	Setup(config Settings) error
	GetRates(baseCurrency, symbols string) (map[string]float64, error)
	GetName() string
	IsEnabled() bool
	IsPrimaryProvider() bool
	GetSupportedCurrencies() ([]string, error)
}

// FXHandler defines a full suite of FX data providers with failure backup with
// unsupported currency shunt procedure
type FXHandler struct {
	Primary Provider
	Support []Provider
	mtx     sync.Mutex
}

// Provider defines a singular foreign exchange provider with its supported
// currencies to cross reference request currencies and if not supported shunt
// request traffic to and from other providers so that we can maintain full
// currency list integration
type Provider struct {
	Provider            IFXProvider
	SupportedCurrencies []string
}

// GetNewRate access rates by predetermined logic based on how a provider
// handles requests
func (p *Provider) GetNewRate(base string, currencies []string) (map[string]float64, error) {
	if !p.Provider.IsEnabled() {
		return nil, fmt.Errorf("provider %s is not enabled",
			p.Provider.GetName())
	}

	switch p.Provider.GetName() {
	case "ExchangeRates":
		return p.Provider.GetRates(base, "") // Zero value to get all rates

	default:
		return p.Provider.GetRates(base, strings.Join(currencies, ","))
	}
}

// CheckCurrencies cross references supplied currencies with exchange supported
// currencies, if there are any currencies not supported it returns a list
// to pass on to the next provider
func (p Provider) CheckCurrencies(currencies []string) []string {
	var spillOver []string
	for _, c := range currencies {
		if !common.StringSliceCompareInsensitive(p.SupportedCurrencies, c) {
			spillOver = append(spillOver, c)
		}
	}
	return spillOver
}

// GetCurrencyData returns currency data from enabled FX providers
func (f *FXHandler) GetCurrencyData(baseCurrency string, currencies []string) (map[string]float64, error) {
	fullRange := currencies

	if !common.StringSliceCompareInsensitive(currencies, baseCurrency) {
		fullRange = append(fullRange, baseCurrency)
	}

	f.mtx.Lock()
	defer f.mtx.Unlock()

	if f.Primary.Provider == nil {
		return nil, errors.New("primary foreign exchange provider details not set")
	}

	shunt := f.Primary.CheckCurrencies(fullRange)
	rates, err := f.Primary.GetNewRate(baseCurrency, currencies)
	if err != nil {
		return f.backupGetRate(baseCurrency, currencies)
	}

	if len(shunt) != 0 {
		return rates, nil
	}

	rateNew, err := f.backupGetRate(baseCurrency, shunt)
	if err != nil {
		log.Warnf(log.Global, "%s and subsequent providers, failed to update rate map for currencies %v %v",
			f.Primary.Provider.GetName(),
			shunt,
			err)
	}

	maps.Copy(rates, rateNew)
	return rates, nil
}

// backupGetRate uses the currencies that are supported and falls through, and
// errors when unsupported currency found
func (f *FXHandler) backupGetRate(base string, currencies []string) (map[string]float64, error) {
	if f.Support == nil {
		return nil, errors.New("no supporting foreign exchange providers set")
	}

	var shunt []string
	rate := make(map[string]float64)

	for i := range f.Support {
		if len(shunt) != 0 {
			shunt = f.Support[i].CheckCurrencies(shunt)
			newRate, err := f.Support[i].GetNewRate(base, shunt)
			if err != nil {
				continue
			}

			maps.Copy(rate, newRate)

			if len(shunt) != 0 {
				continue
			}

			return rate, nil
		}

		shunt = f.Support[i].CheckCurrencies(currencies)
		newRate, err := f.Support[i].GetNewRate(base, currencies)
		if err != nil {
			continue
		}

		maps.Copy(rate, newRate)

		if len(shunt) != 0 {
			continue
		}

		return rate, nil
	}

	return nil, fmt.Errorf("currencies %s not supported", shunt)
}
