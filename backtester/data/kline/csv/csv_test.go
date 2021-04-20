package csv

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

func TestLoadDataCandles(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	_, err := LoadData(
		common.DataCandle,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv"),
		exch,
		gctkline.FifteenMin.Duration(),
		p,
		a)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataTrades(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	_, err := LoadData(
		common.DataTrade,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		exch,
		gctkline.FifteenMin.Duration(),
		p,
		a)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataInvalid(t *testing.T) {
	exch := testExchange
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	_, err := LoadData(
		-1,
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		exch,
		gctkline.FifteenMin.Duration(),
		p,
		a)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("expected '%v' received '%v'", err, common.ErrInvalidDataType)
	}
}
