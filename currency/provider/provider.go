// Package provider utilises foreign exchange API services to manage relational
// FIAT currencies
package provider

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

// Iprovider enforces standard functions for all foreign exchange providers
// supported in GoCryptoTrader
type Iprovider interface {
	Setup(provCfg config.ProviderConfig)
	GetRates(baseCurrency, symbols string) (map[string]float64, error)
}
