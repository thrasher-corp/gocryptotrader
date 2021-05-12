package simulator

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/openware/irix/asset"
	"github.com/openware/irix/bitstamp"
)

func TestSimulate(t *testing.T) {
	b := bitstamp.Bitstamp{}
	b.SetDefaults()
	o, err := b.FetchOrderbook(currency.NewPair(currency.BTC, currency.USD), asset.Spot)
	if err != nil {
		t.Error(err)
	}

	r := o.SimulateOrder(10000000, true)
	t.Log(r.Status)
	r = o.SimulateOrder(2171, false)
	t.Log(r.Status)
}
