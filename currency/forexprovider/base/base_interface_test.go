package base

import (
	"errors"
	"maps"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errCurrencyNotSupported = errors.New("currency not supported")

type MockProvider struct {
	IFXProvider
	value float64
}

func (m *MockProvider) IsEnabled() bool {
	return true
}

func (m *MockProvider) GetName() string {
	return ""
}

func (m *MockProvider) GetRates(baseCurrency, symbols string) (map[string]float64, error) {
	c := map[string]float64{}
	for s := range strings.SplitSeq(symbols, ",") {
		if s == "XRP" && m.value == 1.5 {
			return nil, errCurrencyNotSupported
		}
		if s == "BTC" {
			c[baseCurrency+s] = m.value
			continue
		}
		c[baseCurrency+s] = 1 / (1 + rand.Float64()) //nolint:gosec // Doesn't need to be a strong random number
	}
	return c, nil
}

func TestBackupGetRate(t *testing.T) {
	var f FXHandler
	_, err := f.backupGetRate("", nil)
	assert.ErrorIs(t, err, errNoProvider)
	f.Support = append(f.Support, Provider{
		SupportedCurrencies: []string{"BTC", "ETH", "XRP"},
		Provider: &MockProvider{
			value: 1.5,
		},
	}, Provider{
		SupportedCurrencies: []string{"BTC", "LTC", "XRP"},
		Provider: &MockProvider{
			value: 2.5,
		},
	})
	_, err = f.backupGetRate("", []string{"XRP"})
	assert.ErrorIs(t, err, errCurrencyNotSupported)
	_, err = f.backupGetRate("", []string{"NOTREALCURRENCY"})
	assert.ErrorIs(t, err, errUnsupportedCurrencies)
	f.Support[0].SupportedCurrencies = []string{"BTC", "ETH"}
	r, err := f.backupGetRate("USD", []string{"BTC", "ETH", "LTC", "XRP"})
	assert.NoError(t, err)
	assert.Len(t, r, 4)
	assert.Equal(t, 1.5, r["USDBTC"])
}

type stubProvider struct {
	IFXProvider
	name      string
	rates     map[string]float64
	err       error
	requested []string
}

func (s *stubProvider) IsEnabled() bool { return true }

func (s *stubProvider) GetName() string { return s.name }

func (s *stubProvider) GetRates(_, symbols string) (map[string]float64, error) {
	s.requested = append(s.requested, symbols)
	if s.err != nil {
		return nil, s.err
	}
	return maps.Clone(s.rates), nil
}

func TestGetCurrencyData(t *testing.T) {
	t.Parallel()

	errPrimary := errors.New("primary provider failure")
	errSupport := errors.New("support provider failure")

	t.Run("primary covers request", func(t *testing.T) {
		t.Parallel()
		support := &stubProvider{name: "support", rates: map[string]float64{"USDAUD": 9.9}}
		f := &FXHandler{
			Primary: Provider{
				Provider:            &stubProvider{name: "primary", rates: map[string]float64{"USDAUD": 1.5}},
				SupportedCurrencies: []string{"AUD", "USD"},
			},
			Support: []Provider{{
				Provider:            support,
				SupportedCurrencies: []string{"AUD", "USD"},
			}},
		}
		rates, err := f.GetCurrencyData("USD", []string{"AUD"})
		require.NoError(t, err, "GetCurrencyData must not error")
		assert.Equal(t, map[string]float64{"USDAUD": 1.5}, rates, "rates should come from the primary provider")
		assert.Empty(t, support.requested, "support provider should not be called")
	})

	t.Run("spillover currencies are not requested from support", func(t *testing.T) {
		t.Parallel()
		support := &stubProvider{name: "support", rates: map[string]float64{"USDEUR": 0.9}}
		f := &FXHandler{
			Primary: Provider{
				Provider:            &stubProvider{name: "primary", rates: map[string]float64{"USDAUD": 1.5}},
				SupportedCurrencies: []string{"AUD", "USD"},
			},
			Support: []Provider{{
				Provider:            support,
				SupportedCurrencies: []string{"EUR", "USD"},
			}},
		}
		rates, err := f.GetCurrencyData("USD", []string{"AUD", "EUR"})
		require.NoError(t, err, "GetCurrencyData must not error")
		assert.Empty(t, support.requested, "support provider should not be called")
		assert.Equal(t, map[string]float64{"USDAUD": 1.5}, rates, "rates should come from the primary provider")
	})

	t.Run("primary error is preserved without support", func(t *testing.T) {
		t.Parallel()
		f := &FXHandler{
			Primary: Provider{
				Provider:            &stubProvider{name: "primary", err: errPrimary},
				SupportedCurrencies: []string{"AUD", "USD"},
			},
		}
		rates, err := f.GetCurrencyData("USD", []string{"AUD"})
		assert.Nil(t, rates, "rates should be nil on failure")
		assert.ErrorIs(t, err, errPrimary, "error should preserve the primary provider failure")
		assert.ErrorIs(t, err, errNoProvider, "error should report the missing support providers")
	})

	t.Run("support fallback after primary error", func(t *testing.T) {
		t.Parallel()
		support := &stubProvider{name: "support", rates: map[string]float64{"USDAUD": 1.25}}
		f := &FXHandler{
			Primary: Provider{
				Provider:            &stubProvider{name: "primary", err: errPrimary},
				SupportedCurrencies: []string{"AUD", "USD"},
			},
			Support: []Provider{{
				Provider:            support,
				SupportedCurrencies: []string{"AUD", "USD"},
			}},
		}
		rates, err := f.GetCurrencyData("USD", []string{"AUD"})
		require.NoError(t, err, "GetCurrencyData must not error")
		assert.Equal(t, map[string]float64{"USDAUD": 1.25}, rates, "rates should come from the support provider")
		assert.Equal(t, []string{"AUD"}, support.requested, "support provider should receive the requested currencies")
	})

	t.Run("primary and support errors are preserved", func(t *testing.T) {
		t.Parallel()
		f := &FXHandler{
			Primary: Provider{
				Provider:            &stubProvider{name: "primary", err: errPrimary},
				SupportedCurrencies: []string{"AUD", "USD"},
			},
			Support: []Provider{{
				Provider:            &stubProvider{name: "support", err: errSupport},
				SupportedCurrencies: []string{"AUD", "USD"},
			}},
		}
		_, err := f.GetCurrencyData("USD", []string{"AUD"})
		assert.ErrorIs(t, err, errPrimary, "error should preserve the primary provider failure")
		assert.ErrorIs(t, err, errSupport, "error should preserve the support provider failure")
	})
}
