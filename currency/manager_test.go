package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
		AssetEnabled:  true,
		Available:     spotAvailable,
		Enabled:       spotEnabled,
		RequestFormat: &PairFormat{Uppercase: true},
		ConfigFormat:  &PairFormat{Uppercase: true, Delimiter: "-"},
	}

	futures := &PairStore{
		AssetEnabled:  false,
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
	require.ErrorIs(t, err, asset.ErrNotSupported)

	_, err = p.Get(asset.CoinMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestPairsManagerMatch(t *testing.T) {
	t.Parallel()

	p := &PairsManager{}

	_, err := p.Match("", 1337)
	require.ErrorIs(t, err, ErrSymbolStringEmpty)

	_, err = p.Match("sillyBilly", 1337)
	require.ErrorIs(t, err, errPairMatcherIsNil)

	p = initTest(t)

	_, err = p.Match("sillyBilly", 1337)
	require.ErrorIs(t, err, ErrPairNotFound)

	_, err = p.Match("sillyBilly", asset.Spot)
	require.ErrorIs(t, err, ErrPairNotFound)

	whatIgot, err := p.Match("bTCuSD", asset.Spot)
	require.NoError(t, err)

	whatIwant, err := NewPairFromString("btc-usd")
	if err != nil {
		t.Fatal(err)
	}

	if !whatIgot.Equal(whatIwant) {
		t.Fatal("expected btc-usd")
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
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = p.Store(asset.Futures, nil)
	require.ErrorIs(t, err, errPairStoreIsNil)
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
	require.ErrorIs(t, err, asset.ErrNotSupported)

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
	require.ErrorIs(t, err, asset.ErrNotSupported)

	err = p.StoreFormat(asset.Spot, nil, true)
	require.ErrorIs(t, err, ErrPairFormatIsNil)

	err = p.StoreFormat(asset.Spot, &PairFormat{Delimiter: "~"}, true)
	require.NoError(t, err)

	ps, err := p.Get(asset.Spot)
	if err != nil {
		t.Fatal(err)
	}

	if ps.ConfigFormat.Delimiter != "~" {
		t.Fatal("unexpected value")
	}

	err = p.StoreFormat(asset.Spot, &PairFormat{Delimiter: "/"}, false)
	require.NoError(t, err)

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
	require.ErrorIs(t, err, asset.ErrNotSupported)

	p.Pairs = nil

	ethusdPairs, err := NewPairsFromStrings([]string{"ETH-USD"})
	if err != nil {
		t.Fatal(err)
	}

	err = p.StorePairs(asset.Spot, ethusdPairs, false)
	require.NoError(t, err)

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
	require.NoError(t, err)

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
	require.NoError(t, err)

	err = p.StorePairs(asset.Futures, ethkrwPairs, false)
	require.NoError(t, err)

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

	err := p.DisablePair(asset.Empty, EMPTYPAIR)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "Empty asset should error")

	err = p.DisablePair(asset.Spot, EMPTYPAIR)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty, "Empty pair should error")

	p.Pairs = nil
	err = p.DisablePair(asset.Spot, NewBTCUSD())
	assert.ErrorIs(t, err, ErrPairManagerNotInitialised, "Uninitialised PairManager should error")

	p = initTest(t)
	err = p.DisablePair(asset.CoinMarginedFutures, EMPTYPAIR)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty, "Non-existent asset type should error")

	p.Pairs[asset.Spot] = nil
	err = p.DisablePair(asset.Spot, EMPTYPAIR)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty, "Empty pair store should error")

	p = initTest(t)
	err = p.DisablePair(asset.Spot, NewPair(LTC, USD))
	assert.ErrorIs(t, err, ErrPairNotFound, "Not Enabled pair should error")

	err = p.DisablePair(asset.Spot, NewBTCUSD())
	assert.NoError(t, err, "DisablePair should not error")
}

func TestEnablePair(t *testing.T) {
	t.Parallel()
	p := initTest(t)
	require.ErrorIs(t, p.EnablePair(asset.Empty, NewBTCUSD()), asset.ErrNotSupported)

	p.Pairs = nil
	// Test enabling a pair when the pair manager is not initialised
	if err := p.EnablePair(asset.Spot, NewBTCUSD()); err == nil {
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
	if err := p.EnablePair(asset.Spot, NewBTCUSD()); err == nil {
		t.Error("unexpected result")
	}

	// Test enabling a valid pair
	if err := p.EnablePair(asset.Spot, NewPair(LTC, USD)); err != nil {
		t.Error("unexpected result")
	}
}

func TestAssetEnabled(t *testing.T) {
	t.Parallel()
	p := initTest(t)

	err := p.IsAssetEnabled(asset.Empty)
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	p.Pairs = nil

	// Test enabling a pair when the pair manager is not initialised
	err = p.IsAssetEnabled(asset.Spot)
	assert.ErrorIs(t, err, ErrPairManagerNotInitialised)

	err = p.SetAssetEnabled(asset.Spot, true)
	assert.ErrorIs(t, err, ErrPairManagerNotInitialised)

	// Test asset type which doesn't exist
	p = initTest(t)

	p.Pairs[asset.Spot].AssetEnabled = false

	err = p.IsAssetEnabled(asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotEnabled)

	err = p.SetAssetEnabled(asset.Spot, false)
	assert.NoError(t, err)

	err = p.SetAssetEnabled(asset.Spot, false)
	assert.NoError(t, err, "Setting to disabled twice should not error")

	err = p.IsAssetEnabled(asset.Spot)
	assert.ErrorIs(t, err, asset.ErrNotEnabled)

	err = p.SetAssetEnabled(asset.Spot, true)
	assert.NoError(t, err)

	err = p.SetAssetEnabled(asset.Spot, true)
	assert.NoError(t, err, "Setting to enabled twice should not error")

	err = p.IsAssetEnabled(asset.Spot)
	assert.NoError(t, err, "IsAssetEnabled should not error")
}

// TestFullStoreUnmarshalMarshal tests json Mashal and Unmarshal
func TestFullStoreUnmarshalMarshal(t *testing.T) {
	t.Parallel()
	um := make(FullStore)
	um[asset.Spot] = &PairStore{AssetEnabled: true}

	data, err := json.Marshal(um)
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != `{"spot":{"assetEnabled":true,"enabled":"","available":""}}` {
		t.Fatal("unexpected value")
	}

	var another FullStore
	err = json.Unmarshal(data, &another)
	require.NoError(t, err)

	if _, ok := another[asset.Spot]; !ok {
		t.Fatal("expected values to be associated with spot")
	}

	data = []byte(`{123:{"assetEnabled":null,"enabled":"","available":""}}`)
	err = json.Unmarshal(data, &another)
	assert.Error(t, err, "Unmarshal should error with invalid asset type")

	data = []byte(`{"bro":{"assetEnabled":null,"enabled":"","available":""}}`)
	err = json.Unmarshal(data, &another)
	require.ErrorIs(t, err, asset.ErrNotSupported)
}

func TestIsPairAvailable(t *testing.T) {
	t.Parallel()
	pm := initTest(t)
	cp := NewPairWithDelimiter("BTC", "USD", "-")
	ok, err := pm.IsPairAvailable(cp, asset.Spot)
	require.NoError(t, err, "IsPairAvailable must not error")
	assert.True(t, ok, "IsPairAvailable should return correct value for an available and enabled pair")

	ok, err = pm.IsPairAvailable(NewPair(SAFE, MOONRISE), asset.Spot)
	require.NoError(t, err, "IsPairAvailable must not error")
	assert.False(t, ok, "IsPairAvailable should return correct value for an non-existent")

	ok, err = pm.IsPairAvailable(cp, asset.Futures)
	require.NoError(t, err, "IsPairAvailable must not error")
	assert.False(t, ok, "IsPairAvailable should return false for a disabled asset type")

	cp = NewPairWithDelimiter("XRP", "DOGE", "-")
	ok, err = pm.IsPairAvailable(cp, asset.Spot)
	require.NoError(t, err, "IsPairAvailable must not error")
	assert.False(t, ok, "IsPairAvailable should return false for non-existent pair")

	_, err = pm.IsPairAvailable(cp, asset.PerpetualSwap)
	assert.ErrorIs(t, err, ErrAssetNotFound, "Should error when asset is not found")

	_, err = pm.IsPairAvailable(cp, asset.Item(1337))
	assert.ErrorIs(t, err, asset.ErrNotSupported, "Should error when asset is not supported")

	pm.Pairs[asset.PerpetualSwap] = &PairStore{}
	_, err = pm.IsPairAvailable(cp, asset.PerpetualSwap)
	require.NoError(t, err, "Must not error when store is empty")

	_, err = pm.IsPairAvailable(EMPTYPAIR, asset.PerpetualSwap)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty, "Should error when currency pair is empty")
}

func TestIsPairEnabled(t *testing.T) {
	t.Parallel()
	pm := initTest(t)
	cp := NewPairWithDelimiter("BTC", "USD", "-")
	enabled, err := pm.IsPairEnabled(cp, asset.Spot)
	require.NoError(t, err)
	assert.True(t, enabled, "IsPairEnabled should return true when pair is enabled")

	enabled, err = pm.IsPairEnabled(NewPair(SAFE, MOONRISE), asset.Spot)
	require.NoError(t, err)
	assert.False(t, enabled, "IsPairEnabled should return false when pair does not exist")

	enabled, err = pm.IsPairEnabled(cp, asset.Futures)
	require.NoError(t, err)
	assert.False(t, enabled, "IsPairEnabled should return false when asset not enabled")

	cp = NewPairWithDelimiter("XRP", "DOGE", "-")
	enabled, err = pm.IsPairEnabled(cp, asset.Spot)
	require.NoError(t, err)
	assert.False(t, enabled, "IsPairEnabled should return false when pair not in enabled list")

	_, err = pm.IsPairEnabled(cp, asset.PerpetualSwap)
	assert.ErrorIs(t, err, ErrAssetNotFound)

	_, err = pm.IsPairEnabled(cp, asset.Item(1337))
	assert.ErrorIs(t, err, asset.ErrNotSupported)

	pm.Pairs[asset.PerpetualSwap] = &PairStore{}
	enabled, err = pm.IsPairEnabled(cp, asset.PerpetualSwap)
	require.NoError(t, err, "Must not error when store is empty")
	assert.False(t, enabled, "Should return false when store is empty")

	_, err = pm.IsPairEnabled(EMPTYPAIR, asset.PerpetualSwap)
	assert.ErrorIs(t, err, ErrCurrencyPairEmpty)
}

func TestEnsureOnePairEnabled(t *testing.T) {
	t.Parallel()
	p := NewBTCUSDT()
	pm := PairsManager{
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {},
			asset.Spot: {
				AssetEnabled: true,
				Available: []Pair{
					p,
				},
			},
		},
	}
	pair, item, err := pm.EnsureOnePairEnabled()
	assert.NoError(t, err)

	if len(pm.Pairs[asset.Spot].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Spot].Enabled), 1)
	}
	if item != asset.Spot || !pair.Equal(p) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Spot, p)
	}

	pair, item, err = pm.EnsureOnePairEnabled()
	assert.NoError(t, err)

	if len(pm.Pairs[asset.Spot].Enabled) != 1 {
		t.Errorf("received: '%v' but expected: '%v'", len(pm.Pairs[asset.Spot].Enabled), 1)
	}

	if item != asset.Empty || !pair.Equal(EMPTYPAIR) {
		t.Errorf("received: '%v %v' but expected: '%v %v'", item, p, asset.Empty, EMPTYPAIR)
	}

	pm = PairsManager{
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {
				AssetEnabled: true,
				Available: []Pair{
					NewPair(BTC, USDC),
				},
			},
			asset.Spot: {
				AssetEnabled: true,
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
	assert.NoError(t, err)

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
				AssetEnabled: true,
				Available:    []Pair{p},
			},
			asset.Options: {
				AssetEnabled: true,
				Available:    []Pair{},
			},
		},
	}
	pair, item, err = pm.EnsureOnePairEnabled()
	assert.NoError(t, err)

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
	assert.ErrorIs(t, err, ErrCurrencyPairsEmpty)
}

func TestLoad(t *testing.T) {
	t.Parallel()
	base := PairsManager{}
	fmt1 := &PairFormat{Uppercase: true}
	fmt2 := &PairFormat{Uppercase: true, Delimiter: DashDelimiter}
	p := NewBTCUSDT()
	tt := int64(1337)
	seed := PairsManager{
		LastUpdated:     tt,
		UseGlobalFormat: true,
		ConfigFormat:    fmt1,
		RequestFormat:   fmt2,
		Pairs: map[asset.Item]*PairStore{
			asset.Futures: {
				AssetEnabled: true,
				Available:    []Pair{p},
			},
			asset.Options: {
				AssetEnabled: false,
				Available:    []Pair{},
			},
		},
	}

	base.Load(&seed)
	assert.True(t, base.Pairs[asset.Futures].AssetEnabled, "Futures AssetEnabled should be true")
	assert.True(t, base.Pairs[asset.Futures].Available.Contains(p, true), "Futures Available Pairs should contain BTCUSDT")
	assert.False(t, base.Pairs[asset.Options].AssetEnabled, "Options AssetEnabled should be false")
	assert.Equal(t, tt, base.LastUpdated, "Last Updated should be correct")
	assert.Equal(t, fmt1.Uppercase, base.ConfigFormat.Uppercase, "ConfigFormat Uppercase should be correct")
	assert.Equal(t, fmt2.Delimiter, base.RequestFormat.Delimiter, "RequestFormat Delimiter should be correct")
	found, err := base.Match("BTCUSDT", asset.Futures)
	require.NoError(t, err, "Match must not error")
	assert.Equal(t, p, found, "Should find the right pair")
}

func checkPairDelimiter(tb testing.TB, p *PairsManager, err error, d, msg string) {
	tb.Helper()
	if assert.NoError(tb, err, "UnmarshalJSON should not error") {
		err := p.SetDelimitersFromConfig()
		assert.NoError(tb, err, "SetDelimitersFromConfig should not error")
		s := p.Pairs[asset.Spot]
		if assert.NotNil(tb, s, "Spot asset should not be nil") {
			for _, ps := range []Pairs{s.Enabled, s.Available} {
				if assert.NotEmpty(tb, ps, "PairStore should not be empty") {
					for _, p := range ps {
						assert.Equalf(tb, d, p.Delimiter, msg)
					}
				}
			}
		}
	}
}

// TestPairManagerSetDelimitersFromConfig tests behaviour expectations:
// * Should error with no ConfigFormat ( `CheckPairConsistency` catches that )
// * Asset pair config should take precedent
// * Global store pair config should apply when no Asset pair config
func TestPairManagerSetDelimitersFromConfig(t *testing.T) {
	t.Parallel()
	p := new(PairsManager)

	err := json.Unmarshal([]byte(`{"pairs":{"spot":{"enabled":"BTC-USD,M_PIT-USDT","available":"BTC-USD,M_PIT-USD"}}}`), p)
	if assert.NoError(t, err, "UnmarshalJSON should not error") {
		err = p.SetDelimitersFromConfig()
		assert.ErrorIs(t, err, errPairConfigFormatNil, "SetDelimitersFromConfig should error correctly")
	}

	err = json.Unmarshal([]byte(`{"pairs":{"spot":{"configFormat":{"delimiter":"-"},"enabled":"BTC-USD,M_PIT-USDT","available":"BTC-USD,M_PIT-USD"}}}`), p)
	checkPairDelimiter(t, p, err, "-", "Delimiter should be correct with only bottom level configFormat")

	err = json.Unmarshal([]byte(`{"configFormat":{"delimiter":"/"},"pairs":{"spot":{"enabled":"BTC/USD,M_PIT/USDT","available":"BTC/USD,M_PIT/USD"}}}`), p)
	checkPairDelimiter(t, p, err, "/", "Delimiter should be correct with top level configFormat")

	err = json.Unmarshal([]byte(`{"configFormat":{"delimiter":"_"},"pairs":{"spot":{"configFormat":{"delimiter":"/"},"enabled":"BTC/USD,M_PIT/USDT","available":"BTC/USD,M_PIT/USD"}}}`), p)
	checkPairDelimiter(t, p, err, "/", "Delimiter should be correct with bottom level configFormat")

	err = json.Unmarshal([]byte(`{"pairs":{"spot":{"configFormat":{"delimiter":"_"},"enabled":"BTC-USDT","available":"BTC-USDT"}}}`), p)
	if assert.NoError(t, err, "UnmarshalJSON should not error") {
		err := p.SetDelimitersFromConfig()
		assert.ErrorIs(t, err, errDelimiterNotFound, "SetDelimitersFromConfig should error correctly")
	}
}

// TestGetFormat exercises PairsManager GetFormat
func TestGetFormat(t *testing.T) {
	t.Parallel()

	m := PairsManager{
		UseGlobalFormat: true,
		ConfigFormat: &PairFormat{
			Uppercase: true,
			Delimiter: "🦄",
		},
		RequestFormat: &PairFormat{
			Delimiter: "~",
		},
	}

	pFmt, err := m.GetFormat(asset.Spot, true)
	require.NoError(t, err)
	assert.Equal(t, "~", pFmt.Delimiter, "Global Format Delimiter should be correct")
	assert.False(t, pFmt.Uppercase, "Global Format Uppercase should be correct")

	pFmt, err = m.GetFormat(asset.Spot, false)
	require.NoError(t, err)
	assert.Equal(t, "🦄", pFmt.Delimiter, "Global Format Delimiter should be special")
	assert.True(t, pFmt.Uppercase, "Global Format Uppercase should be correct")

	m.ConfigFormat = nil
	m.RequestFormat = nil
	_, err = m.GetFormat(asset.Spot, true)
	assert.ErrorIs(t, err, ErrPairFormatIsNil, "Global GetFormat Should error correctly on nil request format")
	_, err = m.GetFormat(asset.Spot, false)
	assert.ErrorIs(t, err, ErrPairFormatIsNil, "Global GetFormat Should error correctly on nil config format")

	m.UseGlobalFormat = false
	err = m.Store(asset.Spot, &PairStore{
		ConfigFormat:  &pFmt,
		RequestFormat: &PairFormat{Delimiter: "/", Uppercase: true},
	})
	require.NoError(t, err, "Store must not error")

	pFmt, err = m.GetFormat(asset.Spot, false)
	require.NoError(t, err)
	assert.Equal(t, "🦄", pFmt.Delimiter, "Per Asset Format Delimiter should be correct")
	assert.True(t, pFmt.Uppercase, "Per Asset Format Uppercase should be correct")

	pFmt, err = m.GetFormat(asset.Spot, true)
	require.NoError(t, err)
	assert.Equal(t, "/", pFmt.Delimiter, "Per Asset Format Delimiter should be correct")
	assert.True(t, pFmt.Uppercase, "Per Asset Format Uppercase should be correct")

	err = m.Store(asset.Spot, &PairStore{})
	require.NoError(t, err, "Store must not error")
	_, err = m.GetFormat(asset.Spot, true)
	assert.ErrorIs(t, err, ErrPairFormatIsNil, "Per Asset GetFormat Should error correctly on nil request format")
	_, err = m.GetFormat(asset.Spot, false)
	assert.ErrorIs(t, err, ErrPairFormatIsNil, "Per Asset GetFormat Should error correctly on nil config format")
}

// TestIsAssetSupported exercises IsAssetSupported
func TestIsAssetSupported(t *testing.T) {
	t.Parallel()
	p := PairsManager{
		Pairs: FullStore{
			asset.Spot: {
				AssetEnabled: false,
			},
		},
	}
	assert.True(t, p.IsAssetSupported(asset.Spot), "Spot should be supported")
	assert.False(t, p.IsAssetSupported(asset.Index), "Index should not be supported")
}
