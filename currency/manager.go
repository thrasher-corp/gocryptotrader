package currency

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	// ErrAssetAlreadyEnabled defines an error for the pairs management system
	// that declares the asset is already enabled.
	ErrAssetAlreadyEnabled = errors.New("asset already enabled")
	// ErrPairAlreadyEnabled returns when enabling a pair that is already enabled
	ErrPairAlreadyEnabled = errors.New("pair already enabled")
	// ErrPairNotFound is returned when a currency pair is not found
	ErrPairNotFound = errors.New("pair not found")
	// errAssetNotEnabled defines an error for the pairs management system
	// that declares the asset is not enabled.
	errAssetNotEnabled = errors.New("asset not enabled")
	// ErrAssetIsNil is an error when the asset has not been populated by the
	// configuration
	ErrAssetIsNil = errors.New("asset is nil")
	// ErrPairNotContainedInAvailablePairs defines an error when a pair is not
	// contained in the available pairs list and is not supported by the
	// exchange for that asset type.
	ErrPairNotContainedInAvailablePairs = errors.New("pair not contained in available pairs")

	errPairStoreIsNIl = errors.New("pair store is nil")
)

// GetAssetTypes returns a list of stored asset types
func (p *PairsManager) GetAssetTypes(enabled bool) asset.Items {
	p.m.RLock()
	defer p.m.RUnlock()
	assetTypes := make(asset.Items, 0, len(p.Pairs))
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
			fmt.Errorf("cannot get pair store, %v %w", a, asset.ErrNotSupported)
	}
	return c.copy()
}

// Store stores a new currency pair config based on its asset type
func (p *PairsManager) Store(a asset.Item, ps *PairStore) error {
	cpy, err := ps.copy()
	if err != nil {
		return err
	}
	p.m.Lock()
	if p.Pairs == nil {
		p.Pairs = make(map[asset.Item]*PairStore)
	}
	p.Pairs[a] = cpy
	p.m.Unlock()
	return nil
}

// Delete deletes a map entry based on the supplied asset type
func (p *PairsManager) Delete(a asset.Item) {
	p.m.Lock()
	delete(p.Pairs, a)
	p.m.Unlock()
}

// GetPairs gets a list of stored pairs based on the asset type and whether
// they're enabled or not
func (p *PairsManager) GetPairs(a asset.Item, enabled bool) (Pairs, error) {
	p.m.RLock()
	defer p.m.RUnlock()
	c, ok := p.Pairs[a]
	if !ok {
		return nil, nil
	}

	if !enabled {
		availPairs := make(Pairs, len(c.Available))
		copy(availPairs, c.Available)
		return availPairs, nil
	}

	lenCheck := len(c.Enabled)
	if lenCheck == 0 {
		return nil, nil
	}

	enabledPairs := make(Pairs, lenCheck)
	copy(enabledPairs, c.Enabled)

	err := c.Available.ContainsAll(c.Enabled, true)
	if err != nil {
		return enabledPairs, fmt.Errorf("%w of asset type %s", err, a)
	}
	return enabledPairs, nil
}

// StoreFormat stores a new format for request or config format.
func (p *PairsManager) StoreFormat(a asset.Item, pFmt *PairFormat, config bool) {
	var newCopy *PairFormat
	if pFmt != nil {
		cpy := *pFmt
		newCopy = &cpy
	}

	p.m.Lock()
	defer p.m.Unlock()

	if p.Pairs == nil {
		p.Pairs = make(map[asset.Item]*PairStore)
	}

	c, ok := p.Pairs[a]
	if !ok {
		c = new(PairStore)
		p.Pairs[a] = c
	}

	if config {
		c.ConfigFormat = newCopy
	} else {
		c.RequestFormat = newCopy
	}
}

// StorePairs stores a list of pairs based on the asset type and whether
// they're enabled or not
func (p *PairsManager) StorePairs(a asset.Item, pairs Pairs, enabled bool) {
	cpy := make(Pairs, len(pairs))
	copy(cpy, pairs)

	p.m.Lock()
	defer p.m.Unlock()

	if p.Pairs == nil {
		p.Pairs = make(map[asset.Item]*PairStore)
	}

	c, ok := p.Pairs[a]
	if !ok {
		c = new(PairStore)
		p.Pairs[a] = c
	}

	if enabled {
		c.Enabled = cpy
	} else {
		c.Available = cpy
	}
}

// DisablePair removes the pair from the enabled pairs list if found
func (p *PairsManager) DisablePair(a asset.Item, pair Pair) error {
	p.m.Lock()
	defer p.m.Unlock()

	c, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	enabled, err := c.Enabled.Remove(pair)
	if err != nil {
		if errors.Is(err, ErrPairNotFound) {
			return errors.New("specified pair is not enabled")
		}
		return err
	}
	c.Enabled = enabled
	return nil
}

// EnablePair adds a pair to the list of enabled pairs if it exists in the list
// of available pairs and isn't already added
func (p *PairsManager) EnablePair(a asset.Item, pair Pair) error {
	p.m.Lock()
	defer p.m.Unlock()

	c, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	if c.Enabled.Contains(pair, true) {
		return fmt.Errorf("%s %w", pair, ErrPairAlreadyEnabled)
	}

	if !c.Available.Contains(pair, true) {
		return fmt.Errorf("%s %w in the list of available pairs",
			pair, ErrPairNotFound)
	}
	c.Enabled = c.Enabled.Add(pair)
	return nil
}

// IsAssetEnabled checks to see if an asset is enabled
func (p *PairsManager) IsAssetEnabled(a asset.Item) error {
	p.m.RLock()
	defer p.m.RUnlock()

	c, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	if c.AssetEnabled == nil {
		return fmt.Errorf("%s %w", a, ErrAssetIsNil)
	}

	if !*c.AssetEnabled {
		return fmt.Errorf("%s %w", a, errAssetNotEnabled)
	}
	return nil
}

// SetAssetEnabled sets if an asset is enabled or disabled for first run
func (p *PairsManager) SetAssetEnabled(a asset.Item, enabled bool) error {
	p.m.Lock()
	defer p.m.Unlock()

	c, err := p.getPairStoreRequiresLock(a)
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
		return ErrAssetAlreadyEnabled
	}

	*c.AssetEnabled = enabled
	return nil
}

func (p *PairsManager) getPairStoreRequiresLock(a asset.Item) (*PairStore, error) {
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

// UnmarshalJSON implements the unmarshal json interface so that the key can be
// correctly unmarshalled from a string into a uint.
func (fs *FullStore) UnmarshalJSON(d []byte) error {
	var temp map[string]*PairStore
	err := json.Unmarshal(d, &temp)
	if err != nil {
		return err
	}

	*fs = make(FullStore, len(temp))
	for key, val := range temp {
		ai, err := asset.New(key)
		if err != nil {
			return err
		}
		(*fs)[ai] = val
	}
	return nil
}

// MarshalJSON implements the marshal json interface so that the key can be
// correctly marshalled from a uint.
func (fs FullStore) MarshalJSON() ([]byte, error) {
	temp := make(map[string]*PairStore, len(fs))
	for key, val := range fs {
		temp[key.String()] = val
	}
	return json.Marshal(temp)
}

// copy copies and segregates pair store from internal and external calls.
func (ps *PairStore) copy() (*PairStore, error) {
	if ps == nil {
		return nil, errPairStoreIsNIl
	}
	var assetEnabled *bool
	if ps.AssetEnabled != nil {
		assetEnabled = convert.BoolPtr(*ps.AssetEnabled)
	}

	enabled := make(Pairs, len(ps.Enabled))
	copy(enabled, ps.Enabled)

	avail := make(Pairs, len(ps.Available))
	copy(avail, ps.Available)

	var rFmt *PairFormat
	if ps.RequestFormat != nil {
		cpy := *ps.RequestFormat
		rFmt = &cpy
	}

	var cFmt *PairFormat
	if ps.ConfigFormat != nil {
		cpy := *ps.ConfigFormat
		cFmt = &cpy
	}

	return &PairStore{
		AssetEnabled:  assetEnabled,
		Enabled:       enabled,
		Available:     avail,
		RequestFormat: rFmt,
		ConfigFormat:  cFmt,
	}, nil
}
