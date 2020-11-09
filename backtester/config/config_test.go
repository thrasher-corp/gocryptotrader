package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestButts(t *testing.T) {
	cfg := new(Config)
	cfg.StrategyToLoad = "buyandhold"
	cfg.ExchangeSettings = ExchangeSettings{
		Base:             currency.BTC.String(),
		Quote:            currency.USDT.String(),
		Asset:            asset.Spot.String(),
		MakerFee:         0.01,
		TakerFee:         0.02,
		InitialFunds:     1337,
		MaximumOrderSize: 100,
	}
	cfg.CandleData = &CandleData{
		StartDate: time.Now().Add(-time.Hour * 24 * 7),
		EndDate:   time.Now(),
		Interval:  kline.OneHour.Duration(),
	}
	cfg.DataSource = "candle"
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", result)
}
