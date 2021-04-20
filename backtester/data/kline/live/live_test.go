package live

import (
	"errors"
	"log"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/subsystems/exchangemanager"
)

const testExchange = "FTX"

func TestLoadCandles(t *testing.T) {
	t.Parallel()
	interval := gctkline.FifteenMin
	bot := new(engine.Engine)
	bot.Config = &config.Config{}
	err := bot.Config.LoadConfig(filepath.Join("..", "..", "..", "..", "testdata", "configtest.json"), true)
	if err != nil {
		t.Fatalf("SetupTest: Failed to load config: %s", err)
	}
	bot.ExchangeManager = exchangemanager.Setup()
	err = bot.LoadExchange(testExchange, false, nil)
	if err != nil {
		log.Fatal(err)
	}
	exch := bot.ExchangeManager.GetExchangeByName(testExchange)
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USD)
	var data *gctkline.Item
	data, err = LoadData(exch, common.DataCandle, interval.Duration(), p, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}

	_, err = LoadData(exch, -1, interval.Duration(), p, a)
	if !errors.Is(err, common.ErrInvalidDataType) {
		t.Errorf("expected '%v' received '%v'", err, common.ErrInvalidDataType)
	}
}

func TestLoadTrades(t *testing.T) {
	t.Parallel()
	interval := gctkline.FifteenMin
	bot, err := engine.NewFromSettings(&engine.Settings{
		ConfigFile:   filepath.Join("..", "..", "..", "..", "testdata", "configtest.json"),
		EnableDryRun: true,
	}, nil)
	if err != nil {
		t.Error(err)
	}
	bot.ExchangeManager = exchangemanager.Setup()

	err = bot.LoadExchange(testExchange, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	exch := bot.GetExchangeByName(testExchange)
	if exch == nil {
		t.Fatal("expected binance")
	}
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var data *gctkline.Item
	data, err = LoadData(exch, common.DataTrade, interval.Duration(), p, a)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}
}
