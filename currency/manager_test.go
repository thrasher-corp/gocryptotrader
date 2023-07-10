package currency

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func initTest(t *testing.T) *PairsManager {
	t.Helper()
	spotAvailable, err := NewPairsFromStrings([]string{"BTC-USD", "LTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	spotEnabled, err := NewPairsFromStrings([]string{"BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	spot := &PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		Available:     spotAvailable,
		Enabled:       spotEnabled,
		RequestFormat: &PairFormat{Uppercase: true},
		ConfigFormat:  &PairFormat{Uppercase: true, Delimiter: "-"},
	}

	futures := &PairStore{
		AssetEnabled:  convert.BoolPtr(false),
		Available:     spotAvailable,
		Enabled:       spotEnabled,
		RequestFormat: &PairFormat{Uppercase: true},
		ConfigFormat:  &PairFormat{Uppercase: true, Delimiter: "-"},
	}

	var p PairsManager

	err = p.Store(asset.Spot, spot)
	if err != nil {
		t.Fatal(err)
	}
	err = p.Store(asset.Futures, futures)
	if err != nil {
		t.Fatal(err)
	}

	return &p
}

func TestGetAssetTypes(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	a := p.GetAssetTypes(false)
	if len(a) != 2 {
		t.Errorf("expected 2 but received: %d", len(a))
	}

	a = p.GetAssetTypes(true)
	if len(a) != 1 {
		t.Errorf("GetAssetTypes shouldn't be nil")
	}

	if !a.Contains(asset.Spot) {
		t.Errorf("AssetTypeSpot should be in the assets list")
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	_, err := p.Get(asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = p.Get(asset.Empty)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	_, err = p.Get(asset.CoinMarginedFutures)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}
}

func TestStore(t *testing.T) {
	t.Parallel()
	availPairs, err := NewPairsFromStrings([]string{"BTC-USD", "LTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	enabledPairs, err := NewPairsFromStrings([]string{"BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	p := initTest(t)

	err = p.Store(asset.Futures,
		&PairStore{
			Available: availPairs,
			Enabled:   enabledPairs,
			RequestFormat: &PairFormat{
				Uppercase: true,
			},
			ConfigFormat: &PairFormat{
				Uppercase: true,
				Delimiter: "-",
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	f, err := p.Get(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}

	if f == nil {
		t.Error("Futures assets shouldn't be nil")
	}

	err = p.Store(asset.Empty, nil)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	err = p.Store(asset.Futures, nil)
	if !errors.Is(err, errPairStoreIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errPairStoreIsNil)
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	p.Pairs = nil
	p.Delete(asset.Spot)

	btcusdPairs, err := NewPairsFromStrings([]string{"BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	err = p.Store(asset.Spot, &PairStore{Available: btcusdPairs})
	if err != nil {
		t.Fatal(err)
	}

	p.Delete(asset.UpsideProfitContract)
	spotPS, err := p.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if spotPS == nil {
		t.Error("AssetTypeSpot should exist")
	}

	p.Delete(asset.Spot)

	if _, err := p.Get(asset.Spot); err == nil {
		t.Error("Delete should have deleted AssetTypeSpot")
	}
}

func TestGetPairs(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	p.Pairs = nil
	pairs, err := p.GetPairs(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}

	if pairs != nil {
		t.Fatal("pairs shouldn't be populated")
	}

	p = initTest(t)
	pairs, err = p.GetPairs(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}
	if pairs == nil {
		t.Fatal("pairs should be populated")
	}

	pairs, err = p.GetPairs(asset.Empty, true)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	if pairs != nil {
		t.Fatal("pairs shouldn't be populated")
	}

	superfluous := NewPair(DASH, USDT)
	newPairs := p.Pairs[asset.Spot].Enabled.Add(superfluous)
	p.Pairs[asset.Spot].Enabled = newPairs

	_, err = p.GetPairs(asset.Spot, true)
	if err == nil {
		t.Fatal("error cannot be nil")
	}
}

func TestStoreFormat(t *testing.T) {
	t.Parallel()
	p := &PairsManager{}

	err := p.StoreFormat(0, &PairFormat{Delimiter: "~"}, true)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	err = p.StoreFormat(asset.Spot, nil, true)
	if !errors.Is(err, errPairFormatIsNil) {
		t.Fatalf("received: %v but expected: %v", err, errPairFormatIsNil)
	}

	err = p.StoreFormat(asset.Spot, &PairFormat{Delimiter: "~"}, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
	ps, err := p.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if ps.ConfigFormat.Delimiter != "~" {
		t.Fatal("unexpected value")
	}

	err = p.StoreFormat(asset.Spot, &PairFormat{Delimiter: "/"}, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	ps, err = p.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if ps.RequestFormat.Delimiter != "/" {
		t.Fatal("unexpected value")
	}
}

func TestStorePairs(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	err := p.StorePairs(0, nil, false)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: %v but expected: %v", err, asset.ErrNotSupported)
	}

	p.Pairs = nil

	ethusdPairs, err := NewPairsFromStrings([]string{"ETH-USD"})
	if err != nil {
		t.Fatal(err)
	}

	err = p.StorePairs(asset.Spot, ethusdPairs, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	pairs, err := p.GetPairs(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}

	ethusd, err := NewPairFromString("ETH-USD")
	if err != nil {
		t.Fatal(err)
	}

	if !pairs.Contains(ethusd, true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}

	p = initTest(t)
	err = p.StorePairs(asset.Spot, ethusdPairs, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
	pairs, err = p.GetPairs(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}

	if pairs == nil {
		t.Errorf("pairs should be populated")
	}

	if !pairs.Contains(ethusd, true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}

	ethkrwPairs, err := NewPairsFromStrings([]string{"ETH-KRW"})
	if err != nil {
		t.Error(err)
	}

	err = p.StorePairs(asset.Futures, ethkrwPairs, true)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	err = p.StorePairs(asset.Futures, ethkrwPairs, false)
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}

	pairs, err = p.GetPairs(asset.Futures, true)
	if err != nil {
		t.Fatal(err)
	}

	if pairs == nil {
		t.Errorf("pairs futures should be populated")
	}

	ethkrw, err := NewPairFromString("ETH-KRW")
	if err != nil {
		t.Error(err)
	}

	if !pairs.Contains(ethkrw, true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}
}

func TestDisablePair(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	if err := p.DisablePair(asset.Empty, EMPTYPAIR); !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	if err := p.DisablePair(asset.Spot, EMPTYPAIR); !errors.Is(err, ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrCurrencyPairEmpty)
	}

	p.Pairs = nil
	// Test disabling a pair when the pair manager is not initialised
	if err := p.DisablePair(asset.Spot, NewPair(BTC, USD)); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which doesn't exist
	p = initTest(t)
	if err := p.DisablePair(asset.Futures, EMPTYPAIR); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which has an empty pair store
	p.Pairs[asset.Spot] = nil
	if err := p.DisablePair(asset.Spot, EMPTYPAIR); err == nil {
		t.Error("unexpected result")
	}

	// Test disabling a pair which isn't enabled
	p = initTest(t)
	if err := p.DisablePair(asset.Spot, NewPair(LTC, USD)); err == nil {
		t.Error("unexpected result")
	}

	// Test disabling a valid pair and ensure nil is empty
	if err := p.DisablePair(asset.Spot, NewPair(BTC, USD)); err != nil {
		t.Error("unexpected result")
	}
}

func TestEnablePair(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	if err := p.EnablePair(asset.Empty, NewPair(BTC, USD)); !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	p.Pairs = nil
	// Test enabling a pair when the pair manager is not initialised
	if err := p.EnablePair(asset.Spot, NewPair(BTC, USD)); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which doesn't exist
	p = initTest(t)
	if err := p.EnablePair(asset.Futures, EMPTYPAIR); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which has an empty pair store
	p.Pairs[asset.Spot] = nil
	if err := p.EnablePair(asset.Spot, EMPTYPAIR); err == nil {
		t.Error("unexpected result")
	}

	// Test enabling a pair which isn't in the list of available pairs
	p = initTest(t)
	if err := p.EnablePair(asset.Spot, NewPair(ETH, USD)); err == nil {
		t.Error("unexpected result")
	}

	// Test enabling a pair which already is enabled
	if err := p.EnablePair(asset.Spot, NewPair(BTC, USD)); err == nil {
		t.Error("unexpected result")
	}

	// Test enabling a valid pair
	if err := p.EnablePair(asset.Spot, NewPair(LTC, USD)); err != nil {
		t.Error("unexpected result")
	}
}

func TestIsAssetEnabled_SetAssetEnabled(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	err := p.IsAssetEnabled(asset.Empty)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}
	p.Pairs = nil
	// Test enabling a pair when the pair manager is not initialised
	err = p.IsAssetEnabled(asset.Spot)
	if err == nil {
		t.Error("unexpected result")
	}

	err = p.SetAssetEnabled(asset.Spot, true)
	if err == nil {
		t.Fatal("unexpected result")
	}

	// Test asset type which doesn't exist
	p = initTest(t)

	p.Pairs[asset.Spot].AssetEnabled = nil

	err = p.IsAssetEnabled(asset.Spot)
	if err == nil {
		t.Error("unexpected result")
	}

	err = p.SetAssetEnabled(asset.Spot, false)
	if err != nil {
		t.Fatal(err)
	}

	err = p.SetAssetEnabled(asset.Spot, false)
	if err == nil {
		t.Fatal("unexpected result")
	}

	err = p.IsAssetEnabled(asset.Spot)
	if err == nil {
		t.Error("unexpected result")
	}

	err = p.SetAssetEnabled(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}

	err = p.SetAssetEnabled(asset.Spot, true)
	if err == nil {
		t.Fatal("unexpected result")
	}

	err = p.IsAssetEnabled(asset.Spot)
	if err != nil {
		t.Error("unexpected result")
	}
}

func TestUnmarshalMarshal(t *testing.T) {
	t.Parallel()
	var um = make(FullStore)
	um[asset.Spot] = &PairStore{AssetEnabled: convert.BoolPtr(true)}

	data, err := json.Marshal(um)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != `{"spot":{"assetEnabled":true,"enabled":"","available":""}}` {
		t.Fatal("unexpected value")
	}

	var another FullStore
	err = json.Unmarshal(data, &another)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	if _, ok := another[asset.Spot]; !ok {
		t.Fatal("expected values to be associated with spot")
	}

	data = []byte(`{123:{"assetEnabled":null,"enabled":"","available":""}}`)
	err = json.Unmarshal(data, &another)
	if errors.Is(err, nil) {
		t.Fatalf("expected error")
	}

	data = []byte(`{"bro":{"assetEnabled":null,"enabled":"","available":""}}`)
	err = json.Unmarshal(data, &another)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}
}

func TestIsAssetPairEnabled(t *testing.T) {
	t.Parallel()
	pm := initTest(t)
	cp := NewPairWithDelimiter("BTC", "USD", "-")
	err := pm.IsAssetPairEnabled(asset.Spot, cp)
	if !errors.Is(err, nil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, nil)
	}

	err = pm.IsAssetPairEnabled(asset.Futures, cp)
	if !errors.Is(err, asset.ErrNotEnabled) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotEnabled)
	}

	cp = NewPairWithDelimiter("XRP", "DOGE", "-")
	err = pm.IsAssetPairEnabled(asset.Spot, cp)
	if !errors.Is(err, ErrPairNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrPairNotFound)
	}

	err = pm.IsAssetPairEnabled(asset.PerpetualSwap, cp)
	if !errors.Is(err, errAssetNotFound) {
		t.Fatalf("received: '%v' but expected: '%v'", err, errAssetNotFound)
	}

	err = pm.IsAssetPairEnabled(asset.Item(1337), cp)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Fatalf("received: '%v' but expected: '%v'", err, asset.ErrNotSupported)
	}

	pm.Pairs[asset.PerpetualSwap] = &PairStore{}
	err = pm.IsAssetPairEnabled(asset.PerpetualSwap, cp)
	if !errors.Is(err, ErrAssetIsNil) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrAssetIsNil)
	}

	err = pm.IsAssetPairEnabled(asset.PerpetualSwap, EMPTYPAIR)
	if !errors.Is(err, ErrCurrencyPairEmpty) {
		t.Fatalf("received: '%v' but expected: '%v'", err, ErrCurrencyPairEmpty)
	}
}

func TestEnsureOnePairEnabled(t *testing.T) {
	t.Parallel()
	p := NewPair(BTC, USDT)
	pm := PairsManager{
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {},
			asset.Spot: {
				AssetEnabled: convert.BoolPtr(true),
				Available: []Pair{
					p,
				},
			},
		},
	}
	pair, item, err := pm.EnsureOnePairEnabled()
	if !errors.Is(err, nil) {
		t.Errorf("received: '%v' but expected: '%v'", err, nil)
	}
	if len(pm.Pairs[asset.Spot].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Spot].Enabled), 1)
	}
	if item != asset.Spot || !pair.Equal(p) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Spot, p)
	}

	pair, item, err = pm.EnsureOnePairEnabled()
	if !errors.Is(err, nil) {
		t.Errorf("received: '%v' but expected: '%v'", err, nil)
	}
	if len(pm.Pairs[asset.Spot].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Spot].Enabled), 1)
	}

	if item != asset.Empty || !pair.Equal(EMPTYPAIR) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Empty, EMPTYPAIR)
	}

	pm = PairsManager{
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {
				AssetEnabled: convert.BoolPtr(true),
				Available: []Pair{
					NewPair(BTC, USDC),
				},
			},
			asset.Spot: {
				AssetEnabled: convert.BoolPtr(true),
				Enabled: []Pair{
					p,
				},
				Available: []Pair{
					p,
				},
			},
		},
	}
	pair, item, err = pm.EnsureOnePairEnabled()
	if !errors.Is(err, nil) {
		t.Errorf("received: '%v' but expected: '%v'", err, nil)
	}
	if len(pm.Pairs[asset.Spot].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Spot].Enabled), 1)
	}
	if len(pm.Pairs[asset.Futures].Enabled) != 0 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Futures].Enabled), 0)
	}
	if item != asset.Empty || !pair.Equal(EMPTYPAIR) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Empty, EMPTYPAIR)
	}

	pm = PairsManager{
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {
				AssetEnabled: convert.BoolPtr(true),
				Available:    []Pair{p},
			},
			asset.Options: {
				AssetEnabled: convert.BoolPtr(true),
				Available:    []Pair{},
			},
		},
	}
	pair, item, err = pm.EnsureOnePairEnabled()
	if !errors.Is(err, nil) {
		t.Errorf("received: '%v' but expected: '%v'", err, nil)
	}
	if len(pm.Pairs[asset.Futures].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Futures].Enabled), 1)
	}
	if item != asset.Futures || !pair.Equal(p) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Futures, p)
	}

	pm = PairsManager{
		Pairs: map[asset.Item]*PairStore{},
	}
	_, _, err = pm.EnsureOnePairEnabled()
	if !errors.Is(err, ErrCurrencyPairsEmpty) {
		t.Errorf("received: '%v' but expected: '%v'", err, ErrCurrencyPairsEmpty)
	}
}
