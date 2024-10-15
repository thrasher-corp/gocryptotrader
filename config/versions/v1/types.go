package v1

import v0 "github.com/thrasher-corp/gocryptotrader/config/versions/v0"

// PairsManager contains exchange pair management config
type PairsManager struct {
	BypassConfigFormatUpgrades bool           `json:"bypassConfigFormatUpgrades"`
	RequestFormat              *v0.PairFormat `json:"requestFormat,omitempty"`
	ConfigFormat               *v0.PairFormat `json:"configFormat,omitempty"`
	UseGlobalFormat            bool           `json:"useGlobalFormat,omitempty"`
	LastUpdated                int64          `json:"lastUpdated,omitempty"`
	Pairs                      FullStore      `json:"pairs"`
}

// FullStore contains a pair store by asset name
type FullStore map[string]struct {
	Enabled       string         `json:"enabled"`
	Available     string         `json:"available"`
	RequestFormat *v0.PairFormat `json:"requestFormat,omitempty"`
	ConfigFormat  *v0.PairFormat `json:"configFormat,omitempty"`
}
