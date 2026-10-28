package currency

import (
	"errors"
	"math/rand/v2"
	"strings"
	"sync/atomic"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

var errCountingMockProvider = errors.New("counting mock provider error")

type MockProvider struct{}

func newMockProvider() *forexprovider.ForexProviders {
	p := &MockProvider{}
	c, _ := p.GetSupportedCurrencies()
	return &forexprovider.ForexProviders{
		Primary: base.Provider{
			Provider:            p,
			SupportedCurrencies: c,
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

type CountingMockProvider struct {
	calls atomic.Int64
	fail  atomic.Bool
}

func newCountingMockProvider() (*forexprovider.ForexProviders, *CountingMockProvider) {
	p := &CountingMockProvider{}
	return &forexprovider.ForexProviders{
		Primary: base.Provider{
			Provider:            p,
			SupportedCurrencies: []string{"USD", "EUR", "AUD", "CAD"},
		},
	}, p
}

func (c *CountingMockProvider) GetName() string           { return "CountingMockProvider" }
func (c *CountingMockProvider) Setup(base.Settings) error { return nil }
func (c *CountingMockProvider) IsEnabled() bool           { return true }
func (c *CountingMockProvider) IsPrimaryProvider() bool   { return true }
func (c *CountingMockProvider) GetSupportedCurrencies() ([]string, error) {
	return []string{"USD", "EUR", "AUD", "CAD"}, nil
}

func (c *CountingMockProvider) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	call := c.calls.Add(1)
	if c.fail.Load() {
		return nil, errCountingMockProvider
	}
	res := map[string]float64{}
	for s := range strings.SplitSeq(symbols, ",") {
		// A new rate on every call shows when the cache was refreshed
		res[baseCurrency+s] = float64(call)
	}
	return res, nil
}
