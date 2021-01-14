package csv

import (
	"path/filepath"
	"strings"
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
	_, err := LoadData(filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv"),
		common.CandleStr,
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
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		common.TradeStr,
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
		filepath.Join("..", "..", "..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		"test",
		exch,
		gctkline.FifteenMin.Duration(),
		p,
		a)
	if err != nil && !strings.Contains(err.Error(), "unrecognised csv datatype received") {
		t.Error(err)
	}
}
