package api

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binanceus"

func TestLoadCandles(t *testing.T) {
	t.Parallel()
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	cp := currency.NewPair(currency.BTC, currency.USDT)
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
	tt1 := time.Now().Add(-time.Minute).Round(gctkline.OneMin.Duration())
	tt2 := time.Now().Round(gctkline.OneMin.Duration())
	interval := gctkline.OneMin
	a := asset.Spot
	var data *gctkline.Item
	data, err = LoadData(context.Background(),
		common.DataCandle, tt1, tt2, interval.Duration(), exch, cp, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}

	_, err = LoadData(context.Background(),
		-1, tt1, tt2, interval.Duration(), exch, cp, a)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("received: %v, expected: %v", err, common.ErrInvalidDataType)
	}
}

func TestLoadTrades(t *testing.T) {
	t.Parallel()
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	cp := currency.NewPair(currency.BTC, currency.USDT)
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true},
	}
	interval := gctkline.OneMin
	tt1 := time.Now().Add(-time.Minute * 10).Round(interval.Duration())
	tt2 := time.Now().Round(interval.Duration())
	a := asset.Spot
	var data *gctkline.Item
	data, err = LoadData(context.Background(),
		common.DataTrade, tt1, tt2, interval.Duration(), exch, cp, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}
}
