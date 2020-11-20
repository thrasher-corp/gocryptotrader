package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// these are tests for experimentation more than anything
func TestGenerateCandleAPIConfig(t *testing.T) {
	cfg := Config{
		StrategyToLoad: "dollarcostaverage",
		ExchangeSettings: ExchangeSettings{
			Name:         "binance",
			Asset:        asset.Spot.String(),
			Base:         currency.BTC.String(),
			Quote:        currency.USDT.String(),
			InitialFunds: 1337,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
			MakerFee: 0.01,
			TakerFee: 0.02,
		},
		APIData: &APIData{
			StartDate: time.Now().Add(-time.Hour * 24 * 7),
			EndDate:   time.Now(),
			Interval:  kline.OneHour.Duration(),
			DataType:  common.CandleStr,
		},
		DatabaseData: nil,
		LiveData:     nil,
		PortfolioSettings: PortfolioSettings{
			DiversificationSomething: 0,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
		},
	}
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", result)
}

func TestGenerateCandleLiveConfig(t *testing.T) {
	cfg := Config{
		StrategyToLoad: "dollarcostaverage",
		ExchangeSettings: ExchangeSettings{
			Name:         "binance",
			Asset:        asset.Spot.String(),
			Base:         currency.BTC.String(),
			Quote:        currency.USDT.String(),
			InitialFunds: 1337,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
			MakerFee: 0.01,
			TakerFee: 0.02,
		},
		LiveData: &LiveData{
			Interval: kline.OneMin.Duration(),
			DataType: common.CandleStr,
		},
		PortfolioSettings: PortfolioSettings{
			DiversificationSomething: 0,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
		},
	}
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", result)
}

// these are tests for experimentation more than anything
func TestGenerateRSIAPIConfig(t *testing.T) {
	cfg := Config{
		StrategyToLoad: "rsi420blazeit",
		ExchangeSettings: ExchangeSettings{
			Name:         "binance",
			Asset:        asset.Spot.String(),
			Base:         currency.BTC.String(),
			Quote:        currency.USDT.String(),
			InitialFunds: 1337,
			BuySide: MinMax{
				MinimumSize:  0,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
			MakerFee: 0.01,
			TakerFee: 0.02,
		},
		APIData: &APIData{
			StartDate: time.Now().Add(-time.Hour * 24 * 7),
			EndDate:   time.Now(),
			Interval:  kline.OneHour.Duration(),
			DataType:  common.CandleStr,
		},
		DatabaseData: nil,
		LiveData:     nil,
		PortfolioSettings: PortfolioSettings{
			DiversificationSomething: 0,
			BuySide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			SellSide: MinMax{
				MinimumSize:  0.1,
				MaximumSize:  1,
				MaximumTotal: 10000,
			},
			Leverage: Leverage{
				CanUseLeverage:  false,
				MaximumLeverage: 102,
			},
		},
		StrategySettings: map[string]interface{}{
			"rsi-low":    31.0,
			"rsi-high":   69.0,
			"rsi-period": 12,
		},
	}
	result, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		t.Error(err)
	}
	t.Logf("%s", result)
}
