// Package forexprovider utilises foreign exchange API services to manage
// relational FIAT currencies
package forexprovider

import (
	"time"

	"github.com/thrasher-/gocryptotrader/config"
)

// Base stores the individual provider information
type Base struct {
	Name             string
	Enabled          bool
	Verbose          bool
	RESTPollingDelay time.Duration
	APIKey           string
	APIKeyLvl        int
}

// IFXProvider enforces standard functions for all foreign exchange providers
// supported in GoCryptoTrader
type IFXProvider interface {
	Setup(config config.ForexProviderConfig)
	GetRates(baseCurrency, symbols string) (map[string]float64, error)
}
