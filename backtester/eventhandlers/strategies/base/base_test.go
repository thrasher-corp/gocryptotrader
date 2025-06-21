package base

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	datakline "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/kline"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestGetBase(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	_, err := s.GetBaseData(nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	_, err = s.GetBaseData(datakline.NewDataFromKline())
	assert.ErrorIs(t, err, common.ErrNilEvent)

	tt := time.Now()
	exch := "binance"
	a := asset.Spot
	p := currency.NewBTCUSDT()
	d := &data.Base{}
	err = d.SetStream([]data.Event{&kline.Kline{
		Base: &event.Base{
			Exchange:     exch,
			Time:         tt,
			Interval:     gctkline.OneDay,
			CurrencyPair: p,
			AssetType:    a,
		},
		Open:   decimal.NewFromInt(1337),
		Close:  decimal.NewFromInt(1337),
		Low:    decimal.NewFromInt(1337),
		High:   decimal.NewFromInt(1337),
		Volume: decimal.NewFromInt(1337),
	}})
	assert.NoError(t, err)

	_, err = d.Next()
	assert.NoError(t, err)

	_, err = s.GetBaseData(&datakline.DataFromKline{
		Item:        &gctkline.Item{},
		Base:        d,
		RangeHolder: &gctkline.IntervalRangeHolder{},
	})
	assert.NoError(t, err)
}

func TestSetSimultaneousProcessing(t *testing.T) {
	t.Parallel()
	s := Strategy{}
	is := s.UsingSimultaneousProcessing()
	if is {
		t.Error("expected false")
	}
	s.SetSimultaneousProcessing(true)
	is = s.UsingSimultaneousProcessing()
	if !is {
		t.Error("expected true")
	}
}

func TestCloseAllPositions(t *testing.T) {
	t.Parallel()
	s := &Strategy{}
	_, err := s.CloseAllPositions(nil, nil)
	assert.ErrorIs(t, err, gctcommon.ErrFunctionNotSupported)
}
