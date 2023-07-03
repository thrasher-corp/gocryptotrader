package simulator

import (
	"context"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
)

func TestSimulate(t *testing.T) {
	b := bitstamp.Bitstamp{}
	b.SetDefaults()
	b.Verbose = false
	b.CurrencyPairs = currency.PairsManager{
		UseGlobalFormat: true,
		RequestFormat: &currency.PairFormat{
			Uppercase: true,
		},
		Pairs: map[asset.Item]*currency.PairStore{
			asset.Spot: {
				AssetEnabled: convert.BoolPtr(true),
			},
		},
	}
	o, err := b.FetchOrderbook(context.Background(),
		currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.SimulateOrder(10000000, true)
	if err != nil {
		t.Fatal(err)
	}
	_, err = o.SimulateOrder(2171, false)
	if err != nil {
		t.Fatal(err)
	}
}
