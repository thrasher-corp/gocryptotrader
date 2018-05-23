package base

import (
	"time"
)

// Base enforces standard variables across the provider packages
type Base struct {
	Name             string
	Enabled          bool
	Verbose          bool
	RESTPollingDelay time.Duration
	APIKey           string
	APIKeyLvl        int
	PrimaryProvider  bool
}

// GetName returns name of provider
func (b *Base) GetName() string {
	return b.Name
}

// IsEnabled returns true if enabled
func (b *Base) IsEnabled() bool {
	return b.Enabled
}

// IsPrimaryProvider returns true if primary provider
func (b *Base) IsPrimaryProvider() bool {
	return b.PrimaryProvider
}
