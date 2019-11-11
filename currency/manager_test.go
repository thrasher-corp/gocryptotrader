package currency

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var p PairsManager

func initTest() {
	p.Store(asset.Spot,
		PairStore{
			Available: NewPairsFromStrings([]string{"BTC-USD", "LTC-USD"}),
			Enabled:   NewPairsFromStrings([]string{"BTC-USD"}),
			RequestFormat: &PairFormat{
				Uppercase: true,
			},
			ConfigFormat: &PairFormat{
				Uppercase: true,
				Delimiter: "-",
			},
		},
	)
}

func TestGetAssetTypes(t *testing.T) {
	initTest()

	a := p.GetAssetTypes()
	if len(a) == 0 {
		t.Errorf("GetAssetTypes shouldn't be nil")
	}

	if !a.Contains(asset.Spot) {
		t.Errorf("AssetTypeSpot should be in the assets list")
	}
}

func TestGet(t *testing.T) {
	initTest()

	if p.Get(asset.Spot) == nil {
		t.Error("Spot assets shouldn't be nil")
	}

	if p.Get(asset.Futures) != nil {
		t.Error("Futures should be nil")
	}
}

func TestStore(t *testing.T) {
	p.Store(asset.Futures,
		PairStore{
			Available: NewPairsFromStrings([]string{"BTC-USD", "LTC-USD"}),
			Enabled:   NewPairsFromStrings([]string{"BTC-USD"}),
			RequestFormat: &PairFormat{
				Uppercase: true,
			},
			ConfigFormat: &PairFormat{
				Uppercase: true,
				Delimiter: "-",
			},
		},
	)

	if p.Get(asset.Futures) == nil {
		t.Error("Futures assets shouldn't be nil")
	}
}

func TestDelete(t *testing.T) {
	p.Pairs = nil
	p.Delete(asset.Spot)

	p.Store(asset.Spot,
		PairStore{
			Available: NewPairsFromStrings([]string{"BTC-USD"}),
		},
	)
	p.Delete(asset.UpsideProfitContract)
	if p.Get(asset.Spot) == nil {
		t.Error("AssetTypeSpot should exist")
	}

	p.Delete(asset.Spot)
	if p.Get(asset.Spot) != nil {
		t.Error("Delete should have deleted AssetTypeSpot")
	}
}

func TestGetPairs(t *testing.T) {
	p.Pairs = nil
	pairs := p.GetPairs(asset.Spot, true)
	if pairs != nil {
		t.Fatal("pairs shouldn't be populated")
	}

	initTest()
	pairs = p.GetPairs(asset.Spot, true)
	if pairs == nil {
		t.Fatal("pairs should be populated")
	}

	pairs = p.GetPairs("blah", true)
	if pairs != nil {
		t.Fatal("pairs shouldn't be populated")
	}
}

func TestStorePairs(t *testing.T) {
	p.Pairs = nil
	p.StorePairs(asset.Spot, NewPairsFromStrings([]string{"ETH-USD"}), false)
	pairs := p.GetPairs(asset.Spot, false)
	if !pairs.Contains(NewPairFromString("ETH-USD"), true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}

	initTest()
	p.StorePairs(asset.Spot, NewPairsFromStrings([]string{"ETH-USD"}), false)
	pairs = p.GetPairs(asset.Spot, false)
	if pairs == nil {
		t.Errorf("pairs should be populated")
	}

	if !pairs.Contains(NewPairFromString("ETH-USD"), true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}

	p.StorePairs(asset.Futures, NewPairsFromStrings([]string{"ETH-KRW"}), true)
	pairs = p.GetPairs(asset.Futures, true)
	if pairs == nil {
		t.Errorf("pairs futures should be populated")
	}

	if !pairs.Contains(NewPairFromString("ETH-KRW"), true) {
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
	initTest()
	if err := p.DisablePair(asset.Futures, Pair{}); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which has an empty pair store
	p.Pairs[asset.Spot] = nil
	if err := p.DisablePair(asset.Spot, Pair{}); err == nil {
		t.Error("unexpected result")
	}

	// Test disabling a pair which isn't enabled
	initTest()
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
	initTest()
	if err := p.EnablePair(asset.Futures, Pair{}); err == nil {
		t.Error("unexpected result")
	}

	// Test asset type which has an empty pair store
	p.Pairs[asset.Spot] = nil
	if err := p.EnablePair(asset.Spot, Pair{}); err == nil {
		t.Error("unexpected result")
	}

	// Test enabling a pair which isn't in the list of available pairs
	initTest()
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
