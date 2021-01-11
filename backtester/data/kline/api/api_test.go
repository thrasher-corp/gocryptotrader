package api

import (
	"strings"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestLoadCandles(t *testing.T) {
	tt1 := time.Now().Add(-time.Hour)
	tt2 := time.Now()
	interval := gctkline.FifteenMin
	bot, err := engine.NewFromSettings(&engine.Settings{}, nil)
	if err != nil {
		t.Error(err)
	}

	err = bot.LoadExchange("binance", false, nil)
	if err != nil {
		t.Error(err)
	}
	exch := bot.GetExchangeByName("binance")
	if exch == nil {
		t.Error("expected binance")
	}
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var data *gctkline.Item
	data, err = LoadData(common.CandleStr, tt1, tt2, interval.Duration(), exch, p, a)
	if err != nil {
		t.Error(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}

	_, err = LoadData("", tt1, tt2, interval.Duration(), exch, p, a)
	if err != nil && !strings.Contains(err.Error(), "unrecognised api datatype received") {
		t.Error(err)
	}
}

func TestLoadTrades(t *testing.T) {
	tt1 := time.Now().Add(-time.Hour)
	tt2 := time.Now()
	interval := gctkline.FifteenMin
	bot, err := engine.NewFromSettings(&engine.Settings{}, nil)
	if err != nil {
		t.Error(err)
	}

	err = bot.LoadExchange("binance", false, nil)
	if err != nil {
		t.Error(err)
	}
	exch := bot.GetExchangeByName("binance")
	if exch == nil {
		t.Error("expected binance")
	}
	a := asset.Spot
	p := currency.NewPair(currency.BTC, currency.USDT)
	var data *gctkline.Item
	data, err = LoadData(common.TradeStr, tt1, tt2, interval.Duration(), exch, p, a)
	if err != nil {
		t.Error(err)
	}
	if len(data.Candles) == 0 {
		t.Error("expected candles")
	}

}
