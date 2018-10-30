package base

import (
	"time"
)

// Settings enforces standard variables across the provider packages
type Settings struct {
	Name             string        `json:"name"`
	Enabled          bool          `json:"enabled"`
	Verbose          bool          `json:"verbose"`
	RESTPollingDelay time.Duration `json:"restPollingDelay"`
	APIKey           string        `json:"apiKey"`
	APIKeyLvl        int           `json:"apiKeyLvl"`
	PrimaryProvider  bool          `json:"primaryProvider"`
}

// Base enforces standard variables across the provider packages
type Base struct {
	Settings `json:"settings"`
}

// Name returns name of provider
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
