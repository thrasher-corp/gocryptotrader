package base

import (
	"time"
)

// Base enforces standard variables across the provider packages
type Base struct {
	Settings `json:"settings"`
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

// DefaultTimeOut is the default timeout for foreign exchange providers
const DefaultTimeOut = time.Second * 15

// Settings enforces standard variables across the provider packages
type Settings struct {
	Name            string `json:"name"`
	Enabled         bool   `json:"enabled"`
	Verbose         bool   `json:"verbose"`
	APIKey          string `json:"apiKey"`
	APIKeyLvl       int    `json:"apiKeyLvl"`
	PrimaryProvider bool   `json:"primaryProvider"`
}
