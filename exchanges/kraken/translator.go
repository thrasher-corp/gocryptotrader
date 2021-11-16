package kraken

import (
	"sync"
)

var (
	assetTranslator assetTranslatorStore
)

type assetTranslatorStore struct {
	l      sync.RWMutex
	Assets map[string]string
}

// Seed seeds a currency translation pair
func (a *assetTranslatorStore) Seed(orig, alt string) {
	a.l.Lock()
	if a.Assets == nil {
		a.Assets = make(map[string]string)
	}
	a.Assets[orig] = alt
	a.l.Unlock()
}

// LookupAltname converts a currency into its altname (ZUSD -> USD)
func (a *assetTranslatorStore) LookupAltname(target string) string {
	a.l.RLock()
	alt := a.Assets[target]
	a.l.RUnlock()
	return alt
}

// LookupAltname converts an altname to its original type (USD -> ZUSD)
func (a *assetTranslatorStore) LookupCurrency(target string) string {
	a.l.RLock()
	defer a.l.RUnlock()
	for k, v := range a.Assets {
		if v == target {
			return k
		}
	}
	return ""
}

// Seeded returns whether or not the asset translator has been seeded
func (a *assetTranslatorStore) Seeded() bool {
	a.l.RLock()
	isSeeded := len(a.Assets) > 0
	a.l.RUnlock()
	return isSeeded
}
