package csv

import (
	stdcsv "encoding/csv"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

func TestLoadDataCandles(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	_, err := LoadData(
		common.DataCandle,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv"),
		exch,
		gctkline.FifteenMin.Duration(),
		p,
		a,
		false)
	assert.NoError(t, err)
}

func TestLoadDataTrades(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	_, err := LoadData(
		common.DataTrade,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		exch,
		gctkline.FifteenMin.Duration(),
		p,
		a,
		false)
	assert.NoError(t, err)
}

func TestLoadDataInvalid(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewBTCUSDT()
	_, err := LoadData(
		-1,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		exch,
		gctkline.FifteenMin.Duration(),
		p,
		a,
		false)
	assert.ErrorIs(t, err, common.ErrInvalidDataType)

	_, err = LoadData(
		-1,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		exch,
		gctkline.FifteenMin.Duration(),
		p,
		a,
		true)
	assert.ErrorIs(t, err, errNoUSDData)
}

func TestLoadDataInvalidFieldCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		dataType int64
		contents string
	}{
		{
			name:     "candle",
			dataType: common.DataCandle,
			contents: "1546300800,1,2,3,4\n",
		},
		{
			name:     "trade",
			dataType: common.DataTrade,
			contents: "1546300800,1,2\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "malformed.csv")
			err := os.WriteFile(path, []byte(tt.contents), 0o600)
			require.NoError(t, err, "writing test CSV must not error")

			_, err = LoadData(
				tt.dataType,
				path,
				testExchange,
				gctkline.FifteenMin.Duration(),
				currency.NewBTCUSDT(),
				asset.Spot,
				false)
			assert.ErrorIs(t, err, stdcsv.ErrFieldCount, "malformed CSV should return a field count error")
		})
	}
}
