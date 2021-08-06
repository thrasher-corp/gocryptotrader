package live

import (
	"errors"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

func TestLoadCandles(t *testing.T) {
	t.Parallel()
	interval := gctkline.OneHour
	cp1 := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Spot
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	pFormat := &currency.PairFormat{Uppercase: true}
	b := exch.GetBase()
	exch.SetDefaults()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp1},
		Enabled:       currency.Pairs{cp1},
		AssetEnabled:  convert.BoolPtr(true),
		RequestFormat: pFormat,
		ConfigFormat:  pFormat,
	}
	var data *gctkline.Item
	data, err = LoadData(exch, common.DataCandle, interval.Duration(), cp1, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}
	_, err = LoadData(exch, -1, interval.Duration(), cp1, a)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("expected '%v' received '%v'", err, common.ErrInvalidDataType)
	}
}

func TestLoadTrades(t *testing.T) {
	t.Parallel()
	interval := gctkline.OneMin
	cp1 := currency.NewPair(currency.BTC, currency.USDT)
	a := asset.Spot
	em := engine.SetupExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	if err != nil {
		t.Fatal(err)
	}
	pFormat := &currency.PairFormat{Uppercase: true}
	b := exch.GetBase()
	exch.SetDefaults()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp1},
		Enabled:       currency.Pairs{cp1},
		AssetEnabled:  convert.BoolPtr(true),
		RequestFormat: pFormat,
		ConfigFormat:  pFormat,
	}
	var data *gctkline.Item
	data, err = LoadData(exch, common.DataTrade, interval.Duration(), cp1, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}
}
