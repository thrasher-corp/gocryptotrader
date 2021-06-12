package currency

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetAssetTypes returns a list of stored asset types
func (p *PairsManager) GetAssetTypes(enabled bool) asset.Items {
	p.m.RLock()
	defer p.m.RUnlock()
	var assetTypes asset.Items
	for k, ps := range p.Pairs {
		if enabled && (ps.AssetEnabled == nil || !*ps.AssetEnabled) {
			continue
		}
		assetTypes = append(assetTypes, k)
	}
	return assetTypes
}

// Get gets the currency pair config based on the asset type
func (p *PairsManager) Get(a asset.Item) (*PairStore, error) {
	p.m.RLock()
	defer p.m.RUnlock()
	c, ok := p.Pairs[a]
	if !ok {
		return nil,
			fmt.Errorf("cannot get pair store, asset type %s not supported", a)
	}
	return c, nil
}

// Store stores a new currency pair config based on its asset type
func (p *PairsManager) Store(a asset.Item, ps PairStore) {
	p.m.Lock()
	if p.Pairs == nil {
		p.Pairs = make(map[asset.Item]*PairStore)
	}
	p.Pairs[a] = &ps
	p.m.Unlock()
}

// Delete deletes a map entry based on the supplied asset type
func (p *PairsManager) Delete(a asset.Item) {
	p.m.Lock()
	defer p.m.Unlock()
	if p.Pairs == nil {
		return
	}
	delete(p.Pairs, a)
}

// GetPairs gets a list of stored pairs based on the asset type and whether
// they're enabled or not
func (p *PairsManager) GetPairs(a asset.Item, enabled bool) (Pairs, error) {
	p.m.RLock()
	defer p.m.RUnlock()
	if p.Pairs == nil {
		return nil, nil
	}

	c, ok := p.Pairs[a]
	if !ok {
		return nil, nil
	}

	if enabled {
		for i := range c.Enabled {
			if !c.Available.Contains(c.Enabled[i], true) {
				return c.Enabled,
					fmt.Errorf("enabled pair %s of asset type %s not contained in available list",
						c.Enabled[i],
						a)
			}
		}
		return c.Enabled, nil
	}
	return c.Available, nil
}

// StorePairs stores a list of pairs based on the asset type and whether
// they're enabled or not
func (p *PairsManager) StorePairs(a asset.Item, pairs Pairs, enabled bool) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.Pairs == nil {
		p.Pairs = make(map[asset.Item]*PairStore)
	}

	c, ok := p.Pairs[a]
	if !ok {
		p.Pairs[a] = new(PairStore)
		c = p.Pairs[a]
	}

	if enabled {
		c.Enabled = pairs
	} else {
		c.Available = pairs
	}
}

// DisablePair removes the pair from the enabled pairs list if found
func (p *PairsManager) DisablePair(a asset.Item, pair Pair) error {
	p.m.Lock()
	defer p.m.Unlock()

	c, err := p.getPairStore(a)
	if err != nil {
		return err
	}

	if !c.Enabled.Contains(pair, true) {
		return errors.New("specified pair is not enabled")
	}

	c.Enabled = c.Enabled.Remove(pair)
	return nil
}

// EnablePair adds a pair to the list of enabled pairs if it exists in the list
// of available pairs and isn't already added
func (p *PairsManager) EnablePair(a asset.Item, pair Pair) error {
	p.m.Lock()
	defer p.m.Unlock()

	c, err := p.getPairStore(a)
	if err != nil {
		return err
	}

	if !c.Available.Contains(pair, true) {
		return fmt.Errorf("%s pair was not found in the list of available pairs",
			pair)
	}

	if c.Enabled.Contains(pair, true) {
		return fmt.Errorf("%s pair is already enabled", pair)
	}

	c.Enabled = c.Enabled.Add(pair)
	return nil
}

// IsAssetEnabled checks to see if an asset is enabled
func (p *PairsManager) IsAssetEnabled(a asset.Item) error {
	p.m.RLock()
	defer p.m.RUnlock()

	c, err := p.getPairStore(a)
	if err != nil {
		return err
	}

	if c.AssetEnabled == nil {
		return errors.New("cannot ascertain if asset is enabled, variable is nil")
	}

	if !*c.AssetEnabled {
		return fmt.Errorf("asset %s not enabled", a)
	}
	return nil
}

// SetAssetEnabled sets if an asset is enabled or disabled for first run
func (p *PairsManager) SetAssetEnabled(a asset.Item, enabled bool) error {
	p.m.Lock()
	defer p.m.Unlock()

	c, err := p.getPairStore(a)
	if err != nil {
		return err
	}

	if c.AssetEnabled == nil {
		c.AssetEnabled = convert.BoolPtr(enabled)
		return nil
	}

	if !*c.AssetEnabled && !enabled {
		return errors.New("asset already disabled")
	} else if *c.AssetEnabled && enabled {
		return errors.New("asset already enabled")
	}

	*c.AssetEnabled = enabled
	return nil
}

func (p *PairsManager) getPairStore(a asset.Item) (*PairStore, error) {
	if p.Pairs == nil {
		return nil, errors.New("pair manager not initialised")
	}

	c, ok := p.Pairs[a]
	if !ok {
		return nil, errors.New("asset type not found")
	}

	if c == nil {
		return nil, errors.New("currency store is nil")
	}

	return c, nil
}
