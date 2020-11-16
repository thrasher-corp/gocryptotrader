package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestGenerateAPIDCAConfig(t *testing.T) {
	cfg := Config{
		StrategyToLoad: "dollarcostaverage",
		ExchangeSettings: ExchangeSettings{
			Name:            "binance",
			Asset:           asset.Spot.String(),
			Base:            currency.BTC.String(),
			Quote:           currency.USDT.String(),
			InitialFunds:    1337,
			MinimumBuySize:  0.1,
			MaximumBuySize:  1,
			DefaultBuySize:  0.5,
			MinimumSellSize: 0.1,
			MaximumSellSize: 2,
			DefaultSellSize: 0.5,
			CanUseLeverage:  false,
			MaximumLeverage: 0,
			MakerFee:        0.01,
			TakerFee:        0.02,
		},
		APIData: &APIData{
			StartDate: time.Now().Add(-time.Hour * 24 * 7),
			EndDate:   time.Now(),
			Interval:  kline.OneHour.Duration(),
		},
		DatabaseData: nil,
		LiveData:     nil,
		PortfolioSettings: PortfolioSettings{
			DiversificationSomething: 0,
			CanUseLeverage:           false,
			MaximumLeverage:          0,
			MinimumBuySize:           0.1,
			MaximumBuySize:           1,
			DefaultBuySize:           0.5,
			MinimumSellSize:          0.1,
			MaximumSellSize:          2,
			DefaultSellSize:          0.5,
		},
	}
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", result)
}
