package live

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binanceus"

func TestLoadCandles(t *testing.T) {
	t.Parallel()
	interval := gctkline.OneHour
	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Spot
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	pFormat := &currency.PairFormat{Uppercase: true}
	b := exch.GetBase()
	exch.SetDefaults()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		RequestFormat: pFormat,
		ConfigFormat:  pFormat,
	}
	var data *gctkline.Item
	data, err = LoadData(context.Background(), time.Now().Add(-interval.Duration()*10), exch, common.DataCandle, interval.Duration(), cp, currency.EMPTYPAIR, a, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}
	_, err = LoadData(context.Background(), time.Now(), exch, -1, interval.Duration(), cp, currency.EMPTYPAIR, a, true)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("received: %v, expected: %v", err, common.ErrInvalidDataType)
	}
}

func TestLoadTrades(t *testing.T) {
	t.Parallel()
	interval := gctkline.OneMin
	cp := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Spot
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	pFormat := &currency.PairFormat{Uppercase: true}
	b := exch.GetBase()
	exch.SetDefaults()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		RequestFormat: pFormat,
		ConfigFormat:  pFormat,
	}
	var data *gctkline.Item
	data, err = LoadData(context.Background(), time.Now().Add(-interval.Duration()*10), exch, common.DataTrade, interval.Duration(), cp, currency.EMPTYPAIR, a, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}
}
