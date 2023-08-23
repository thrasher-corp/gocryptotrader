package key

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

func TestGenerateMapKey(t *testing.T) {
	t.Parallel()
	_, err := GeneratePairAssetKey(currency.EMPTYPAIR, 0)
	if !errors.Is(err, currency.ErrCurrencyPairEmpty) {
		t.Error(err)
	}

	cp := currency.NewPair(currency.BTC, currency.USDT)
	_, err = GeneratePairAssetKey(cp, 0)
	if !errors.Is(err, asset.ErrInvalidAsset) {
		t.Error(err)
	}

	k, err := GeneratePairAssetKey(cp, asset.Spot)
	if !errors.Is(err, nil) {
		t.Error(err)
	}
	if k.Base != cp.Base.Item {
		t.Errorf("received %v expected %v", k.Base, cp.Base.Item)
	}
	if k.Quote != cp.Quote.Item {
		t.Errorf("received %v expected %v", k.Quote, cp.Quote.Item)
	}
	if k.Asset != asset.Spot {
		t.Errorf("received %v expected %v", k.Asset, asset.Spot)
	}
}
