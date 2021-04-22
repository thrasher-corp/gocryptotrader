package api

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchangemanager "github.com/thrasher-corp/gocryptotrader/engine/exchangemanager"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

var (
	bot  *engine.Engine
	exch exchange.IBotExchange
)

func TestMain(m *testing.M) {
	var err error
	bot, err = engine.NewFromSettings(&engine.Settings{
		ConfigFile:   filepath.Join("..", "..", "..", "..", "testdata", "configtest.json"),
		EnableDryRun: true,
	}, nil)
	if err != nil {
		log.Fatal(err)
	}

	bot.ExchangeManager = exchangemanager.Setup()
	err = bot.LoadExchange(testExchange, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	exch = bot.GetExchangeByName(testExchange)
	if exch == nil {
		log.Fatal("expected binance")
	}
	os.Exit(m.Run())
}

func TestLoadCandles(t *testing.T) {
	t.Parallel()
	tt1 := time.Now().Add(-time.Hour).Round(gctkline.OneHour.Duration())
	tt2 := time.Now().Round(gctkline.OneHour.Duration())
	interval := gctkline.OneHour
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var data *gctkline.Item
	var err error
	data, err = LoadData(common.DataCandle, tt1, tt2, interval.Duration(), exch, p, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}

	_, err = LoadData(-1, tt1, tt2, interval.Duration(), exch, p, a)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("expected '%v' received '%v'", err, common.ErrInvalidDataType)
	}
}

func TestLoadTrades(t *testing.T) {
	t.Parallel()
	interval := gctkline.FifteenMin
	tt1 := time.Now().Add(-time.Minute * 15).Round(interval.Duration())
	tt2 := time.Now().Round(interval.Duration())
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var err error
	var data *gctkline.Item
	data, err = LoadData(common.DataTrade, tt1, tt2, interval.Duration(), exch, p, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}
}
