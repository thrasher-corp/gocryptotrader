package currency

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/exchanges/assets"
)

var p PairsManager

func initTest() {
	p.Store(assets.AssetTypeSpot,
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
		t.Errorf("Test failed. GetAssetTypes shouldn't be nil")
	}

	if !a.Contains(assets.AssetTypeSpot) {
		t.Errorf("Test failed. AssetTypeSpot should be in the assets list")
	}
}

func TestGet(t *testing.T) {
	initTest()

	if p.Get(assets.AssetTypeSpot) == nil {
		t.Error("Test failed. Spot assets shouldn't be nil")
	}

	if p.Get(assets.AssetTypeFutures) != nil {
		t.Error("Test Failed. Futures should be nil")
	}
}

func TestStore(t *testing.T) {
	p.Store(assets.AssetTypeFutures,
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

	if p.Get(assets.AssetTypeFutures) == nil {
		t.Error("Test failed. Futures assets shouldn't be nil")
	}
}

func TestDelete(t *testing.T) {
	p.Pairs = nil
	p.Delete(assets.AssetTypeSpot)

	p.Store(assets.AssetTypeSpot,
		PairStore{
			Available: NewPairsFromStrings([]string{"BTC-USD"}),
		},
	)
	p.Delete(assets.AssetTypeUpsideProfitContract)
	if p.Get(assets.AssetTypeSpot) == nil {
		t.Error("Test failed. AssetTypeSpot should exist")
	}

	p.Delete(assets.AssetTypeSpot)
	if p.Get(assets.AssetTypeSpot) != nil {
		t.Error("Test failed. Delete should have deleted AssetTypeSpot")
	}

}

func TestGetPairs(t *testing.T) {
	p.Pairs = nil
	pairs := p.GetPairs(assets.AssetTypeSpot, true)
	if pairs != nil {
		t.Fatal("pairs shouldn't be populated")
	}

	initTest()
	pairs = p.GetPairs(assets.AssetTypeSpot, true)
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
	p.StorePairs(assets.AssetTypeSpot, NewPairsFromStrings([]string{"ETH-USD"}), false)
	pairs := p.GetPairs(assets.AssetTypeSpot, false)
	if !pairs.Contains(NewPairFromString("ETH-USD"), true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}

	initTest()
	p.StorePairs(assets.AssetTypeSpot, NewPairsFromStrings([]string{"ETH-USD"}), false)
	pairs = p.GetPairs(assets.AssetTypeSpot, false)
	if pairs == nil {
		t.Errorf("pairs should be populated")
	}

	if !pairs.Contains(NewPairFromString("ETH-USD"), true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}

	p.StorePairs(assets.AssetTypeFutures, NewPairsFromStrings([]string{"ETH-KRW"}), true)
	pairs = p.GetPairs(assets.AssetTypeFutures, true)
	if pairs == nil {
		t.Errorf("pairs futures should be populated")
	}

	if !pairs.Contains(NewPairFromString("ETH-KRW"), true) {
		t.Errorf("TestStorePairs failed, unexpected result")
	}
}
