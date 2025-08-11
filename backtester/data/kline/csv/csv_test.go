package csv

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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
