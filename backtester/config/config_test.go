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
	hi := new(Config)
	hi.StrategyToLoad = "buyandhold"
	hi.ExchangePairSettings = make(map[string][]Currency)
	hi.ExchangePairSettings["binance"] = []Currency{
		{
			Base:             currency.BTC.String(),
			Quote:            currency.USDT.String(),
			Asset:            asset.Spot.String(),
			MakerFee:         0.01,
			TakerFee:         0.02,
			InitialFunds:     1337,
			MaximumOrderSize: 100,
		},
	}
	hi.StartDate = time.Now().Add(-time.Hour * 24 * 7)
	hi.EndDate = time.Now()
	hi.DataSource = "candle"
	hi.CandleData = &CandleData{Interval: kline.OneHour.Duration()}
	result, err := json.MarshalIndent(hi, "", " ")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", result)
}
