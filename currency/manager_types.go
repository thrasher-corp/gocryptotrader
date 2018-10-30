package currency

import (
	"sync"

	"github.com/thrasher-/gocryptotrader/exchanges/assets"
)

// PairsManager manages asset pairs
type PairsManager struct {
	RequestFormat   *PairFormat                     `json:"requestFormat,omitempty"`
	ConfigFormat    *PairFormat                     `json:"configFormat,omitempty"`
	UseGlobalFormat bool                            `json:"useGlobalFormat,omitempty"`
	LastUpdated     int64                           `json:"lastUpdated,omitempty"`
	AssetTypes      assets.AssetTypes               `json:"assetTypes"`
	Pairs           map[assets.AssetType]*PairStore `json:"pairs"`
	m               sync.Mutex
}

// PairStore stores a currency pair store
type PairStore struct {
	Enabled       Pairs       `json:"enabled,omitempty"`
	Available     Pairs       `json:"available,omitempty"`
	RequestFormat *PairFormat `json:"requestFormat,omitempty"`
	ConfigFormat  *PairFormat `json:"configFormat,omitempty"`
}

// PairFormat returns the pair format
type PairFormat struct {
	Uppercase bool   `json:"uppercase"`
	Delimiter string `json:"delimiter,omitempty"`
	Separator string `json:"separator,omitempty"`
	Index     string `json:"index,omitempty"`
}
