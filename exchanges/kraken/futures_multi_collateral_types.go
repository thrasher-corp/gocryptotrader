package kraken

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// FuturesPNLCurrencyPreferencesResponse holds the currency each multi-collateral contract pays realised profit in
type FuturesPNLCurrencyPreferencesResponse struct {
	Preferences []FuturesPNLCurrencyPreference `json:"preferences"`
	ServerTime  time.Time                      `json:"serverTime"`
}

// FuturesPNLCurrencyPreference is the currency a contract pays realised profit in
type FuturesPNLCurrencyPreference struct {
	Symbol      string        `json:"symbol"`
	PNLCurrency currency.Code `json:"pnlCurrency"`
}

// FuturesSetPreferenceResponse holds the server time of a preference change
type FuturesSetPreferenceResponse struct {
	ServerTime time.Time `json:"serverTime"`
}

// FuturesLeverageSettingsResponse holds the configured leverage preferences
type FuturesLeverageSettingsResponse struct {
	LeveragePreferences []FuturesLeverageSetting `json:"leveragePreferences"`
	ServerTime          time.Time                `json:"serverTime"`
}

// FuturesLeverageSetting is a contract's maximum leverage
type FuturesLeverageSetting struct {
	Symbol          string  `json:"symbol"`
	MaximumLeverage float64 `json:"maxLeverage"`
}

// FuturesLeverageSettingRequest holds the parameters of Set leverage settings
type FuturesLeverageSettingRequest struct {
	Symbol currency.Pair
	// MaximumLeverage sets the contract to isolated margin at this maximum leverage; zero sets it to cross margin
	MaximumLeverage float64
}
