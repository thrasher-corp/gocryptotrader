package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	cp := currency.NewBTCUSDT()
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
	data, err := LoadData(t.Context(), common.DataCandle, tt1, tt2, interval.Duration(), exch, cp, a)
	require.NoError(t, err, "LoadData must not error")
	assert.NotEmpty(t, data.Candles, "Candles should not be empty")
	_, err = LoadData(t.Context(), -1, tt1, tt2, interval.Duration(), exch, cp, a)
	assert.ErrorIs(t, err, common.ErrInvalidDataType)
}

func TestLoadTrades(t *testing.T) {
	t.Parallel()
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err, "NewExchangeByName must not error")
	exch.SetDefaults()
	cp := currency.NewBTCUSDT()
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
	data, err := LoadData(t.Context(), common.DataTrade, tt1, tt2, interval.Duration(), exch, cp, a)
	require.NoError(t, err, "LoadData must not error")
	assert.NotEmpty(t, data.Candles, "Candles should not be empty")
}
