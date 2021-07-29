package currency

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var p PairsManager

func initTest(t *testing.T) {
	spotAvailable, err := NewPairsFromStrings([]string{"BTC-USD", "LTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	spotEnabled, err := NewPairsFromStrings([]string{"BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	spot := PairStore{
		AssetEnabled:  convert.BoolPtr(true),
		Available:     spotAvailable,
		Enabled:       spotEnabled,
		RequestFormat: &PairFormat{Uppercase: true},
		ConfigFormat:  &PairFormat{Uppercase: true, Delimiter: "-"},
	}

	futures := PairStore{
		AssetEnabled:  convert.BoolPtr(false),
		Available:     spotAvailable,
		Enabled:       spotEnabled,
		RequestFormat: &PairFormat{Uppercase: true},
		ConfigFormat:  &PairFormat{Uppercase: true, Delimiter: "-"},
	}

	p.Store(asset.Spot, spot)
	p.Store(asset.Futures, futures)
}

func TestGetAssetTypes(t *testing.T) {
	initTest(t)

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
	initTest(t)

	_, err := p.Get(asset.Spot)
	if err != nil {
		t.Error(err)
	}

	_, err = p.Get(asset.CoinMarginedFutures)
	if err == nil {
		t.Error("CoinMarginedFutures should be nil")
	}
}

func TestStore(t *testing.T) {
	availPairs, err := NewPairsFromStrings([]string{"BTC-USD", "LTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	enabledPairs, err := NewPairsFromStrings([]string{"BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	p.Store(asset.Futures,
		PairStore{
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

	f, err := p.Get(asset.Futures)
	if err != nil {
		t.Fatal(err)
	}

	if f == nil {
		t.Error("Futures assets shouldn't be nil")
	}
}

func TestDelete(t *testing.T) {
	p.Pairs = nil
	p.Delete(asset.Spot)

	btcusdPairs, err := NewPairsFromStrings([]string{"BTC-USD"})
	if err != nil {
		t.Fatal(err)
	}

	p.Store(asset.Spot, PairStore{
		Available: btcusdPairs,
	})

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
	p.Pairs = nil
	pairs, err := p.GetPairs(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}

	if pairs != nil {
		t.Fatal("pairs shouldn't be populated")
	}

	initTest(t)
	pairs, err = p.GetPairs(asset.Spot, true)
	if err != nil {
		t.Fatal(err)
	}
	if pairs == nil {
		t.Fatal("pairs should be populated")
	}

	pairs, err = p.GetPairs("blah", true)
	if err != nil {
		t.Fatal(err)
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

func TestStorePairs(t *testing.T) {
	p.Pairs = nil

	ethusdPairs, err := NewPairsFromStrings([]string{"ETH-USD"})
	if err != nil {
		t.Fatal(err)
	}

	p.StorePairs(asset.Spot, ethusdPairs, false)
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

	initTest(t)
	p.StorePairs(asset.Spot, ethusdPairs, false)
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

	p.StorePairs(asset.Futures, ethkrwPairs, true)
	p.StorePairs(asset.Futures, ethkrwPairs, false)
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
	p.Pairs = nil
	// Test disabling a pair when the pair manager is not initialised
	if err := p.DisablePair(asset.Spot, NewPair(BTC, USD)); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which doesn't exist
	initTest(t)
	if err := p.DisablePair(asset.Futures, Pair{}); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which has an empty pair store
	p.Pairs[asset.Spot] = nil
	if err := p.DisablePair(asset.Spot, Pair{}); err == nil {
		t.Error("unexpected result")
	}

	// Test disabling a pair which isn't enabled
	initTest(t)
	if err := p.DisablePair(asset.Spot, NewPair(LTC, USD)); err == nil {
		t.Error("unexpected result")
	}

	// Test disabling a valid pair and ensure nil is empty
	if err := p.DisablePair(asset.Spot, NewPair(BTC, USD)); err != nil {
		t.Error("unexpected result")
	}
}

func TestEnablePair(t *testing.T) {
	p.Pairs = nil
	// Test enabling a pair when the pair manager is not initialised
	if err := p.EnablePair(asset.Spot, NewPair(BTC, USD)); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which doesn't exist
	initTest(t)
	if err := p.EnablePair(asset.Futures, Pair{}); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which has an empty pair store
	p.Pairs[asset.Spot] = nil
	if err := p.EnablePair(asset.Spot, Pair{}); err == nil {
		t.Error("unexpected result")
	}

	// Test enabling a pair which isn't in the list of available pairs
	initTest(t)
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
	p.Pairs = nil
	// Test enabling a pair when the pair manager is not initialised
	err := p.IsAssetEnabled(asset.Spot)
	if err == nil {
		t.Error("unexpected result")
	}

	err = p.SetAssetEnabled(asset.Spot, true)
	if err == nil {
		t.Fatal("unexpected result")
	}

	// Test asset type which doesn't exist
	initTest(t)

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
