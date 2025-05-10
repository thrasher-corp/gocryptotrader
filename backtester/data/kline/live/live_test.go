package live

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

const testExchange = "okx"

func TestLoadCandles(t *testing.T) {
	t.Parallel()
	interval := gctkline.OneHour
	cp := currency.NewBTCUSDT()
	a := asset.Spot
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err, "NewExchangeByName must not error")
	pFormat := &currency.PairFormat{Uppercase: true}
	b := exch.GetBase()
	exch.SetDefaults()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		RequestFormat: pFormat,
		ConfigFormat:  pFormat,
	}
	data, err := LoadData(t.Context(), time.Now().Add(-interval.Duration()*10), exch, common.DataCandle, interval.Duration(), cp, currency.EMPTYPAIR, a, true)
	require.NoError(t, err, "LoadData must not error")
	assert.NotEmpty(t, data.Candles, "Candles should not be empty")
	_, err = LoadData(t.Context(), time.Now(), exch, -1, interval.Duration(), cp, currency.EMPTYPAIR, a, true)
	assert.ErrorIs(t, err, common.ErrInvalidDataType)
}

func TestLoadTrades(t *testing.T) {
	t.Parallel()
	interval := gctkline.OneMin
	cp := currency.NewBTCUSDT()
	a := asset.Spot
	em := engine.NewExchangeManager()
	exch, err := em.NewExchangeByName(testExchange)
	require.NoError(t, err, "NewExchangeByName must not error")
	pFormat := &currency.PairFormat{Uppercase: true}
	b := exch.GetBase()
	exch.SetDefaults()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  true,
		RequestFormat: pFormat,
		ConfigFormat:  pFormat,
	}
	data, err := LoadData(t.Context(), time.Now().Add(-interval.Duration()*60), exch, common.DataTrade, interval.Duration(), cp, currency.EMPTYPAIR, a, true)
	require.NoError(t, err, "LoadData must not error")
	assert.NotEmpty(t, data.Candles, "Candles should not be empty")
}
