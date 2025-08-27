package currency

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// Public errors
var (
	ErrAssetNotFound                    = errors.New("asset type not found in pair store")
	ErrPairAlreadyEnabled               = errors.New("pair already enabled")
	ErrPairFormatIsNil                  = errors.New("pair format is nil")
	ErrPairManagerNotInitialised        = errors.New("pair manager not initialised")
	ErrPairNotContainedInAvailablePairs = errors.New("pair not contained in available pairs")
	ErrPairNotEnabled                   = errors.New("pair not enabled")
	ErrPairNotFound                     = errors.New("pair not found")
	ErrSymbolStringEmpty                = errors.New("symbol string is empty")
)

var (
	errPairStoreIsNil      = errors.New("pair store is nil")
	errPairMatcherIsNil    = errors.New("pair matcher is nil")
	errPairConfigFormatNil = errors.New("pair config format is nil")
)

// GetAssetTypes returns a list of stored asset types
func (p *PairsManager) GetAssetTypes(enabled bool) asset.Items {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	assetTypes := make(asset.Items, 0, len(p.Pairs))
	for k, ps := range p.Pairs {
		if enabled && !ps.AssetEnabled {
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

	p.mutex.RLock()
	defer p.mutex.RUnlock()
	c, ok := p.Pairs[a]
	if !ok {
		return nil, fmt.Errorf("cannot get pair store, %v %w", a, asset.ErrNotSupported)
	}
	return c.clone(), nil
}

// Match returns a currency pair based on the supplied symbol and asset type
func (p *PairsManager) Match(symbol string, a asset.Item) (Pair, error) {
	if symbol == "" {
		return EMPTYPAIR, ErrSymbolStringEmpty
	}
	symbol = strings.ToLower(symbol)
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if p.matcher == nil {
		return EMPTYPAIR, errPairMatcherIsNil
	}
	pair, ok := p.matcher[key{symbol, a}]
	if !ok {
		return EMPTYPAIR, fmt.Errorf("%w for %v %v", ErrPairNotFound, symbol, a)
	}
	return *pair, nil
}

// Store stores a new currency pair config based on its asset type
func (p *PairsManager) Store(a asset.Item, ps *PairStore) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}
	if ps == nil {
		return errPairStoreIsNil
	}
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.Pairs == nil {
		p.Pairs = FullStore{}
	}
	p.Pairs[a] = ps.clone()
	p.reindex()
	return nil
}

// Delete deletes a map entry based on the supplied asset type
func (p *PairsManager) Delete(a asset.Item) {
	p.mutex.Lock()
	vals, ok := p.Pairs[a]
	if !ok {
		p.mutex.Unlock()
		return
	}
	for x := range vals.Available {
		delete(p.matcher, key{Symbol: vals.Available[x].Base.Lower().String() + vals.Available[x].Quote.Lower().String(), Asset: a})
	}
	delete(p.Pairs, a)
	p.mutex.Unlock()
}

// GetPairs gets a list of stored pairs based on the asset type and whether
// they're enabled or not
func (p *PairsManager) GetPairs(a asset.Item, enabled bool) (Pairs, error) {
	if !a.IsValid() {
		return nil, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()
	pairStore, ok := p.Pairs[a]
	if !ok {
		return nil, nil
	}

	if !enabled {
		return slices.Clone(pairStore.Available), nil
	}

	lenCheck := len(pairStore.Enabled)
	if lenCheck == 0 {
		return nil, nil
	}

	// NOTE: enabledPairs is declared before the next check for comparison
	// reasons within exchange update pairs functionality.
	enabledPairs := slices.Clone(pairStore.Enabled)

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
		return ErrPairFormatIsNil
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.Pairs == nil {
		p.Pairs = make(map[asset.Item]*PairStore)
	}

	pairStore, ok := p.Pairs[a]
	if !ok {
		pairStore = new(PairStore)
		p.Pairs[a] = pairStore
	}

	if config {
		pairStore.ConfigFormat = pFmt.clone()
	} else {
		pairStore.RequestFormat = pFmt.clone()
	}
	return nil
}

// GetFormat returns the pair format in a concurrent safe manner
func (p *PairsManager) GetFormat(a asset.Item, request bool) (PairFormat, error) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	var pFmt *PairFormat
	if p.UseGlobalFormat {
		if request {
			pFmt = p.RequestFormat
		} else {
			pFmt = p.ConfigFormat
		}
	} else {
		ps, err := p.getPairStoreRequiresLock(a)
		if err != nil {
			return EMPTYFORMAT, err
		}
		if request {
			pFmt = ps.RequestFormat
		} else {
			pFmt = ps.ConfigFormat
		}
	}
	if pFmt == nil {
		return EMPTYFORMAT, ErrPairFormatIsNil
	}
	return *pFmt, nil
}

// StorePairs stores a list of pairs for an asset type
// If enabled is true:
// * AssetEnabled is set true if the pair list is not empty
// * pairs replace the Enabled pairs
// * pairs are added to Available pairs
func (p *PairsManager) StorePairs(a asset.Item, pairs Pairs, enabled bool) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.Pairs == nil {
		p.Pairs = make(map[asset.Item]*PairStore)
	}

	pairStore, ok := p.Pairs[a]
	if !ok {
		pairStore = new(PairStore)
		p.Pairs[a] = pairStore
	}

	if enabled {
		if len(pairs) != 0 {
			pairStore.AssetEnabled = true
			pairStore.Available.Add(pairs...)
		}
		pairStore.Enabled = slices.Clone(pairs)
	} else {
		pairStore.Available = slices.Clone(pairs)
		p.reindex()
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
	p.mutex.Lock()
	defer p.mutex.Unlock()
	for _, v := range p.Pairs {
		if !v.AssetEnabled || len(v.Available) == 0 {
			continue
		}
		if len(v.Enabled) > 0 {
			return EMPTYPAIR, asset.Empty, nil
		}
	}
	for k, v := range p.Pairs {
		if !v.AssetEnabled || len(v.Available) == 0 {
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

	p.mutex.Lock()
	defer p.mutex.Unlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	enabledLen := len(pairStore.Enabled)

	pairStore.Enabled = pairStore.Enabled.Remove(pair)

	if enabledLen == len(pairStore.Enabled) {
		return fmt.Errorf("%w %s", ErrPairNotFound, pair)
	}

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

	p.mutex.Lock()
	defer p.mutex.Unlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	if pairStore.Enabled.Contains(pair, true) {
		return fmt.Errorf("%s %w", pair, ErrPairAlreadyEnabled)
	}

	if !pairStore.Available.Contains(pair, true) {
		return fmt.Errorf("%s %w in the list of available pairs", pair, ErrPairNotFound)
	}
	pairStore.Enabled = pairStore.Enabled.Add(pair)
	return nil
}

// IsPairAvailable checks if a pair is available for a given asset type
func (p *PairsManager) IsPairAvailable(pair Pair, a asset.Item) (bool, error) {
	if !a.IsValid() {
		return false, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	if pair.IsEmpty() {
		return false, ErrCurrencyPairEmpty
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return false, err
	}
	return pairStore.AssetEnabled && pairStore.Available.Contains(pair, true), nil
}

// IsPairEnabled checks if a pair is enabled for an enabled asset type
func (p *PairsManager) IsPairEnabled(pair Pair, a asset.Item) (bool, error) {
	if !a.IsValid() {
		return false, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	if pair.IsEmpty() {
		return false, ErrCurrencyPairEmpty
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return false, err
	}
	return pairStore.AssetEnabled && pairStore.Enabled.Contains(pair, true), nil
}

// IsAssetEnabled checks to see if an asset is enabled
func (p *PairsManager) IsAssetEnabled(a asset.Item) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	p.mutex.RLock()
	defer p.mutex.RUnlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	if !pairStore.AssetEnabled {
		return fmt.Errorf("%s %w", a, asset.ErrNotEnabled)
	}
	return nil
}

// IsAssetSupported returns if the asset is supported by an exchange
// Does not imply that the Asset is enabled
func (p *PairsManager) IsAssetSupported(a asset.Item) bool {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	_, ok := p.Pairs[a]
	return ok
}

// SetAssetEnabled sets if an asset is enabled or disabled for first run
func (p *PairsManager) SetAssetEnabled(a asset.Item, enabled bool) error {
	if !a.IsValid() {
		return fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	pairStore, err := p.getPairStoreRequiresLock(a)
	if err != nil {
		return err
	}

	pairStore.AssetEnabled = enabled

	return nil
}

// Load sets the pair manager from a seed without copying mutexes
func (p *PairsManager) Load(seed *PairsManager) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	seed.mutex.RLock()
	defer seed.mutex.RUnlock()

	p.BypassConfigFormatUpgrades = seed.BypassConfigFormatUpgrades
	p.UseGlobalFormat = seed.UseGlobalFormat
	p.LastUpdated = seed.LastUpdated
	p.Pairs = seed.Pairs.clone()
	p.RequestFormat = seed.RequestFormat.clone()
	p.ConfigFormat = seed.ConfigFormat.clone()
	p.reindex()
}

// reindex re-indexes the matcher for Available pairs and all assets
// This method does not lock for concurrency
func (p *PairsManager) reindex() {
	p.matcher = make(map[key]*Pair)
	for a, fs := range p.Pairs {
		for i, pair := range fs.Available {
			k := key{Symbol: pair.Base.Lower().String() + pair.Quote.Lower().String(), Asset: a}
			p.matcher[k] = &fs.Available[i]
		}
	}
}

func (p *PairsManager) getPairStoreRequiresLock(a asset.Item) (*PairStore, error) {
	if p.Pairs == nil {
		return nil, fmt.Errorf("%w when requesting %v pairs", ErrPairManagerNotInitialised, a)
	}

	pairStore, ok := p.Pairs[a]
	if !ok {
		return nil, fmt.Errorf("%w %w %q", ErrAssetNotFound, asset.ErrNotSupported, a)
	}

	if pairStore == nil {
		return nil, errors.New("currency store is nil")
	}

	return pairStore, nil
}

// SetDelimitersFromConfig ensures that the pairs adhere to the configured delimiters
// Pairs.Unmarshal doesn't know what the delimiter is, so uses the first punctuation rune
func (p *PairsManager) SetDelimitersFromConfig() error {
	for a, s := range p.Pairs {
		cf := s.ConfigFormat
		if cf == nil {
			cf = p.ConfigFormat
		}
		if cf == nil {
			return errPairConfigFormatNil
		}
		for i, p := range []Pairs{s.Enabled, s.Available} {
			for j := range p {
				if cf.Delimiter == "" || p[j].Delimiter == cf.Delimiter {
					continue
				}
				nP, err := NewPairDelimiter(p[j].String(), cf.Delimiter)
				if err != nil {
					return fmt.Errorf("%s.%s.%s: %w", a, []string{"enabled", "available"}[i], p[j], err)
				}
				p[j] = nP
			}
		}
	}
	return nil
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

// clone returns a deep clone of the PairStore
func (ps *PairStore) clone() *PairStore {
	if ps == nil {
		return nil
	}

	return &PairStore{
		AssetEnabled:  ps.AssetEnabled,
		Enabled:       slices.Clone(ps.Enabled),
		Available:     slices.Clone(ps.Available),
		RequestFormat: ps.RequestFormat.clone(),
		ConfigFormat:  ps.ConfigFormat.clone(),
	}
}

func (fs FullStore) clone() FullStore {
	c := FullStore{}
	for a, pairStore := range fs {
		c[a] = pairStore.clone()
	}
	return c
}
