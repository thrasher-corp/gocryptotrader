package currency

import (
	"math/rand"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

type MockProvider struct{}

func newMockProvider() *forexprovider.ForexProviders {
	p := &MockProvider{}
	c, _ := p.GetSupportedCurrencies()
	return &forexprovider.ForexProviders{
		FXHandler: base.FXHandler{
			Primary: base.Provider{
				Provider:            p,
				SupportedCurrencies: c,
			},
		},
	}
}

func (m *MockProvider) GetName() string           { return "MockProvider" }
func (m *MockProvider) Setup(base.Settings) error { return nil }
func (m *MockProvider) IsEnabled() bool           { return true }
func (m *MockProvider) IsPrimaryProvider() bool   { return true }
func (m *MockProvider) GetSupportedCurrencies() ([]string, error) {
	return storage.defaultFiatCurrencies.Strings(), nil
}

func (m *MockProvider) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	c := map[string]float64{}
	for s := range strings.SplitSeq(symbols, ",") {
		// The year is 2027; The USD is nearly worthless. The world reserve currency is eggs.
		c[baseCurrency+s] = 1 / (1 + rand.Float64()) //nolint:gosec // Doesn't need to be a strong random number
	}
	return c, nil
}
