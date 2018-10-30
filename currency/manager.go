package currency

import (
	"github.com/thrasher-/gocryptotrader/exchanges/assets"
)

// GetAssetTypes returns a list of stored asset types
func (p *PairsManager) GetAssetTypes() assets.AssetTypes {
	p.m.Lock()
	defer p.m.Unlock()
	var assetTypes assets.AssetTypes
	for k := range p.Pairs {
		assetTypes = append(assetTypes, k)
	}
	return assetTypes
}

// Get gets the currency pair config based on the asset type
func (p *PairsManager) Get(a assets.AssetType) *PairStore {
	p.m.Lock()
	defer p.m.Unlock()
	c, ok := p.Pairs[a]
	if !ok {
		return nil
	}
	return c
}

// Store stores a new currency pair config based on its asset type
func (p *PairsManager) Store(a assets.AssetType, ps PairStore) {
	p.m.Lock()

	if p.Pairs == nil {
		p.Pairs = make(map[assets.AssetType]*PairStore)
	}

	if !p.AssetTypes.Contains(a) {
		p.AssetTypes = append(p.AssetTypes, a)
	}

	p.Pairs[a] = &ps
	p.m.Unlock()
}

// Delete deletes a map entry based on the supplied asset type
func (p *PairsManager) Delete(a assets.AssetType) {
	p.m.Lock()
	defer p.m.Unlock()
	if p.Pairs == nil {
		return
	}

	_, ok := p.Pairs[a]
	if !ok {
		return
	}

	delete(p.Pairs, a)
}

// GetPairs gets a list of stored pairs based on the asset type and whether
// they're enabled or not
func (p *PairsManager) GetPairs(a assets.AssetType, enabled bool) Pairs {
	p.m.Lock()
	defer p.m.Unlock()
	if p.Pairs == nil {
		return nil
	}

	c, ok := p.Pairs[a]
	if !ok {
		return nil
	}

	var pairs Pairs
	if enabled {
		pairs = c.Enabled
	} else {
		pairs = c.Available
	}

	return pairs
}

// StorePairs stores a list of pairs based on the asset type and whether
// they're enabled or not
func (p *PairsManager) StorePairs(a assets.AssetType, pairs Pairs, enabled bool) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.Pairs == nil {
		p.Pairs = make(map[assets.AssetType]*PairStore)
	}

	c, ok := p.Pairs[a]
	if !ok {
		c = new(PairStore)
	}

	if enabled {
		c.Enabled = pairs
	} else {
		c.Available = pairs
	}

	p.Pairs[a] = c
}
