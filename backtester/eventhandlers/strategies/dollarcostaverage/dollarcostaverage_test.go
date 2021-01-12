package dollarcostaverage

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	eventkline "github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestName(t *testing.T) {
	d := Strategy{}
	n := d.Name()
	if n != Name {
		t.Errorf("expected %v", Name)
	}
}

func TestSupportsMultiCurrency(t *testing.T) {
	d := Strategy{}
	if !d.SupportsMultiCurrency() {
		t.Error("expected true")
	}
}

func TestSetCustomSettings(t *testing.T) {
	d := Strategy{}
	err := d.SetCustomSettings(nil)
	if err != nil && err.Error() != "unsupported" {
		t.Error(err)
	}
	if err == nil {
		t.Error("expected unsupported")
	}
}

func TestOnSignal(t *testing.T) {
	s := Strategy{}
	_, err := s.OnSignal(nil, nil)
	if err != nil && err.Error() != "received nil data" {
		t.Error(err)
	}

	tt := time.Now()
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := data.Data{}
	d.SetStream([]common.DataEventHandler{&eventkline.Kline{
		Event: event.Event{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   1337,
		Close:  1337,
		Low:    1337,
		High:   1337,
		Volume: 1337,
	}})
	d.Next()
	_, err = s.OnSignal(&kline.DataFromKline{
		Item:  gctkline.Item{},
		Data:  d,
		Range: gctkline.IntervalRangeHolder{},
	}, nil)
	if err != nil {
		t.Error(err)
	}
}
