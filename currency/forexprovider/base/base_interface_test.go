package base

import (
	"errors"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
