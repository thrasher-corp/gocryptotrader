package live

import (
	"strings"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

const testExchange = "binance"

func TestLoadCandles(t *testing.T) {
	interval := gctkline.FifteenMin
	bot, err := engine.NewFromSettings(&engine.Settings{}, nil)
	if err != nil {
		t.Error(err)
	}

	err = bot.LoadExchange(testExchange, false, nil)
	if err != nil {
		t.Error(err)
	}
	exch := bot.GetExchangeByName(testExchange)
	if exch == nil {
		t.Error("expected binance")
	}
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var data *gctkline.Item
	data, err = LoadData(exch, common.CandleStr, interval.Duration(), p, a)
	if err != nil {
		t.Error(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}

	_, err = LoadData(exch, "", interval.Duration(), p, a)
	if err != nil && !strings.Contains(err.Error(), "unrecognised api datatype received") {
		t.Error(err)
	}
}

func TestLoadTrades(t *testing.T) {
	interval := gctkline.FifteenMin
	bot, err := engine.NewFromSettings(&engine.Settings{}, nil)
	if err != nil {
		t.Error(err)
	}

	err = bot.LoadExchange(testExchange, false, nil)
	if err != nil {
		t.Error(err)
	}
	exch := bot.GetExchangeByName(testExchange)
	if exch == nil {
		t.Error("expected binance")
	}
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var data *gctkline.Item
	data, err = LoadData(exch, common.TradeStr, interval.Duration(), p, a)
	if err != nil {
		t.Error(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}
}
