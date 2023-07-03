package currency

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/common"
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
	// ErrAssetIsNil is an error when the asset has not been populated by the
	// configuration
	ErrAssetIsNil = errors.New("asset is nil")
	// ErrPairNotContainedInAvailablePairs defines an error when a pair is not
	// contained in the available pairs list and is not supported by the
	// exchange for that asset type.
	ErrPairNotContainedInAvailablePairs = errors.New("pair not contained in available pairs")

	errPairStoreIsNil  = errors.New("pair store is nil")
	errPairFormatIsNil = errors.New("pair format is nil")
	errAssetNotFound   = errors.New("asset type not found")
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
	if !a.IsValid() {
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

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
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
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
	if !a.IsValid() {
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	p.m.RLock()
	defer p.m.RUnlock()
	pairStore, ok := p.Pairs[a]
	if !ok {
		return nil, nil
	}

	if !enabled {
		availPairs := make(Pairs, len(pairStore.Available))
		copy(availPairs, pairStore.Available)
		return availPairs, nil
	}

	lenCheck := len(pairStore.Enabled)
	if lenCheck == 0 {
		return nil, nil
	}

	// NOTE: enabledPairs is declared before the next check for comparison
	// reasons within exchange update pairs functionality.
	enabledPairs := make(Pairs, lenCheck)
	copy(enabledPairs, pairStore.Enabled)

	err := pairStore.Available.ContainsAll(pairStore.Enabled, true)
	if err != nil {
		err = fmt.Errorf("%w of asset type %s", err, a)
	}
	return enabledPairs, err
}

// StoreFormat stores a new format for request or config format.
func (p *PairsManager) StoreFormat(a asset.Item, pFmt *PairFormat, config bool) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
	if pFmt == nil {
		return errPairFormatIsNil
	}

	cpy := *pFmt

	p.m.Lock()
	defer p.m.Unlock()

	if p.Pairs == nil {
		p.Pairs = make(map[asset.Item]*PairStore)
	}

	pairStore, ok := p.Pairs[a]
	if !ok {
		pairStore = new(PairStore)
		p.Pairs[a] = pairStore
	}

	if config {
		pairStore.ConfigFormat = &cpy
	} else {
		pairStore.RequestFormat = &cpy
	}
	return nil
}

// StorePairs stores a list of pairs based on the asset type and whether
// they're enabled or not
func (p *PairsManager) StorePairs(a asset.Item, pairs Pairs, enabled bool) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	// NOTE: Length check not needed in this scenario as it has the ability to
	// remove the entire stored list if needed.
	cpy := make(Pairs, len(pairs))
	copy(cpy, pairs)

	p.m.Lock()
	defer p.m.Unlock()

	if p.Pairs == nil {
		p.Pairs = make(map[asset.Item]*PairStore)
	}

	pairStore, ok := p.Pairs[a]
	if !ok {
		pairStore = new(PairStore)
		p.Pairs[a] = pairStore
	}

	if enabled {
		pairStore.Enabled = cpy
	} else {
		pairStore.Available = cpy
	}

	return nil
}

// EnsureOnePairEnabled not all assets have pairs, eg options
// search for an asset that does and enable one if none are enabled
// error if no currency pairs found for an entire exchange
// returns the asset and pair of a pair if it has been enabled
func (p *PairsManager) EnsureOnePairEnabled() (Pair, asset.Item, error) {
	if p == nil {
		return EMPTYPAIR, asset.Empty, common.ErrNilPointer
	}
	p.m.Lock()
	defer p.m.Unlock()
	for _, v := range p.Pairs {
		if v.AssetEnabled == nil ||
			!*v.AssetEnabled ||
			len(v.Available) == 0 {
			continue
		}
		if len(v.Enabled) > 0 {
			return EMPTYPAIR, asset.Empty, nil
		}
	}
	for k, v := range p.Pairs {
		if v.AssetEnabled == nil ||
			!*v.AssetEnabled ||
			len(v.Available) == 0 {
			continue
		}
		rp, err := v.Available.GetRandomPair()
		if err != nil {
			return EMPTYPAIR, asset.Empty, err
		}
		p.Pairs[k].Enabled = v.Enabled.Add(rp)
		return rp, k, nil
	}
	return EMPTYPAIR, asset.Empty, ErrCurrencyPairsEmpty
}

// DisablePair removes the pair from the enabled pairs list if found
func (p *PairsManager) DisablePair(a asset.Item, pair Pair) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	if pair.IsEmpty() {
		return ErrCurrencyPairEmpty
	}

	p.m.Lock()
	defer p.m.Unlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	enabled, err := pairStore.Enabled.Remove(pair)
	if err != nil {
		return err
	}
	pairStore.Enabled = enabled
	return nil
}

// EnablePair adds a pair to the list of enabled pairs if it exists in the list
// of available pairs and isn't already added
func (p *PairsManager) EnablePair(a asset.Item, pair Pair) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	if pair.IsEmpty() {
		return ErrCurrencyPairEmpty
	}

	p.m.Lock()
	defer p.m.Unlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	if pairStore.Enabled.Contains(pair, true) {
		return fmt.Errorf("%s %w", pair, ErrPairAlreadyEnabled)
	}

	if !pairStore.Available.Contains(pair, true) {
		return fmt.Errorf("%s %w in the list of available pairs",
			pair, ErrPairNotFound)
	}
	pairStore.Enabled = pairStore.Enabled.Add(pair)
	return nil
}

// IsAssetPairEnabled checks if a pair is enabled for an enabled asset type
func (p *PairsManager) IsAssetPairEnabled(a asset.Item, pair Pair) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	if pair.IsEmpty() {
		return ErrCurrencyPairEmpty
	}

	p.m.RLock()
	defer p.m.RUnlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	if pairStore.AssetEnabled == nil {
		return fmt.Errorf("%s %w", a, ErrAssetIsNil)
	}
	if !*pairStore.AssetEnabled {
		return fmt.Errorf("%s %w", a, asset.ErrNotEnabled)
	}
	if !pairStore.Enabled.Contains(pair, true) {
		return fmt.Errorf("%s %w", pair, ErrPairNotFound)
	}

	return nil
}

// IsAssetEnabled checks to see if an asset is enabled
func (p *PairsManager) IsAssetEnabled(a asset.Item) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	p.m.RLock()
	defer p.m.RUnlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	if pairStore.AssetEnabled == nil {
		return fmt.Errorf("%s %w", a, ErrAssetIsNil)
	}

	if !*pairStore.AssetEnabled {
		return fmt.Errorf("%s %w", a, asset.ErrNotEnabled)
	}
	return nil
}

// SetAssetEnabled sets if an asset is enabled or disabled for first run
func (p *PairsManager) SetAssetEnabled(a asset.Item, enabled bool) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	p.m.Lock()
	defer p.m.Unlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	if pairStore.AssetEnabled == nil {
		pairStore.AssetEnabled = convert.BoolPtr(enabled)
		return nil
	}

	if !*pairStore.AssetEnabled && !enabled {
		return errors.New("asset already disabled")
	} else if *pairStore.AssetEnabled && enabled {
		return ErrAssetAlreadyEnabled
	}

	*pairStore.AssetEnabled = enabled
	return nil
}

func (p *PairsManager) getPairStoreRequiresLock(a asset.Item) (*PairStore, error) {
	if p.Pairs == nil {
		return nil, errors.New("pair manager not initialised")
	}

	pairStore, ok := p.Pairs[a]
	if !ok {
		return nil, fmt.Errorf("%w %v", errAssetNotFound, a)
	}

	if pairStore == nil {
		return nil, errors.New("currency store is nil")
	}

	return pairStore, nil
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
		return nil, errPairStoreIsNil
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
