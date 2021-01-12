package base

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestGetBase(t *testing.T) {
	s := Strategy{}
	_, err := s.GetBase(nil)
	if err != nil && err.Error() != "nil data handler received" {
		t.Error(err)
	}

	_, err = s.GetBase(&datakline.DataFromKline{})
	if err != nil && err.Error() != "could not retrieve latest data for strategy" {
		t.Error(err)
	}
	tt := time.Now()
	exch := "binance"
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	d := data.Data{}
	d.SetStream([]common.DataEventHandler{&kline.Kline{
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
	_, err = s.GetBase(&datakline.DataFromKline{
		Item:  gctkline.Item{},
		Data:  d,
		Range: gctkline.IntervalRangeHolder{},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestSetMultiCurrency(t *testing.T) {
	s := Strategy{}
	is := s.IsMultiCurrency()
	if is {
		t.Error("expected false")
	}
	s.SetMultiCurrency(true)
	is = s.IsMultiCurrency()
	if !is {
		t.Error("expected true")
	}
}
