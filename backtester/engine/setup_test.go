package engine

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/dollarcostaverage"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func TestNewFromConfig(t *testing.T) {
	t.Parallel()
	_, err := NewFromConfig(nil, "", "", false)
	if !errors.Is(err, errNilConfig) {
		t.Errorf("received %v, expected %v", err, errNilConfig)
	}

	cfg := &config.Config{}
	_, err = NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("received: %v, expected: %v", err, base.ErrStrategyNotFound)
	}

	cfg.CurrencySettings = []config.CurrencySettings{
		{
			ExchangeName: "test",
			Base:         currency.NewCode("test"),
			Quote:        currency.NewCode("test"),
		},
		{
			ExchangeName: testExchange,
			Base:         currency.BTC,
			Quote:        currency.NewCode("0624"),
			Asset:        asset.Futures,
		},
	}
	_, err = NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, engine.ErrExchangeNotFound) {
		t.Errorf("received: %v, expected: %v", err, engine.ErrExchangeNotFound)
	}
	cfg.CurrencySettings[0].ExchangeName = testExchange
	_, err = NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, asset.ErrNotSupported) {
		t.Errorf("received: %v, expected: %v", err, asset.ErrNotSupported)
	}
	cfg.CurrencySettings[0].Asset = asset.Spot
	_, err = NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, base.ErrStrategyNotFound) {
		t.Errorf("received: %v, expected: %v", err, base.ErrStrategyNotFound)
	}

	cfg.StrategySettings = config.StrategySettings{
		Name: dollarcostaverage.Name,
		CustomSettings: map[string]interface{}{
			"hello": "moto",
		},
	}
	cfg.CurrencySettings[0].Base = currency.BTC
	cfg.CurrencySettings[0].Quote = currency.USD
	cfg.DataSettings.APIData = &config.APIData{
		StartDate: time.Time{},
		EndDate:   time.Time{},
	}

	_, err = NewFromConfig(cfg, "", "", false)
	if err != nil && !strings.Contains(err.Error(), "unrecognised dataType") {
		t.Error(err)
	}
	cfg.DataSettings.DataType = common.CandleStr
	_, err = NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, errIntervalUnset) {
		t.Errorf("received: %v, expected: %v", err, errIntervalUnset)
	}
	cfg.DataSettings.Interval = gctkline.OneMin
	cfg.CurrencySettings[0].MakerFee = &decimal.Zero
	cfg.CurrencySettings[0].TakerFee = &decimal.Zero
	_, err = NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, gctcommon.ErrDateUnset) {
		t.Errorf("received: %v, expected: %v", err, gctcommon.ErrDateUnset)
	}

	cfg.DataSettings.APIData.StartDate = time.Now().Add(-time.Minute)
	cfg.DataSettings.APIData.EndDate = time.Now()
	cfg.DataSettings.APIData.InclusiveEndDate = true
	_, err = NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, holdings.ErrInitialFundsZero) {
		t.Errorf("received: %v, expected: %v", err, holdings.ErrInitialFundsZero)
	}

	cfg.FundingSettings.UseExchangeLevelFunding = true
	cfg.FundingSettings.ExchangeLevelFunding = []config.ExchangeLevelFunding{
		{
			ExchangeName: testExchange,
			Asset:        asset.Spot,
			Currency:     currency.BTC,
			InitialFunds: leet,
			TransferFee:  leet,
		},
		{
			ExchangeName: testExchange,
			Asset:        asset.Futures,
			Currency:     currency.BTC,
			InitialFunds: leet,
			TransferFee:  leet,
		},
	}
	_, err = NewFromConfig(cfg, "", "", false)
	if !errors.Is(err, nil) {
		t.Errorf("received: %v, expected: %v", err, nil)
	}
}

func TestLoadDataAPI(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: "Binance",
				Asset:        asset.Spot,
				Base:         cp.Base,
				Quote:        cp.Quote,
				SpotDetails: &config.SpotDetails{
					InitialQuoteFunds: &leet,
				},
				BuySide:  config.MinMax{},
				SellSide: config.MinMax{},
				MakerFee: &decimal.Zero,
				TakerFee: &decimal.Zero,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin,
			APIData: &config.APIData{
				StartDate: time.Now().Add(-time.Minute * 5),
				EndDate:   time.Now(),
			}},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]interface{}{
				"hello": "moto",
			},
		},
	}
	em := engine.ExchangeManager{}
	exch, err := em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}

	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataDatabase(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: "Binance",
				Asset:        asset.Spot,
				Base:         cp.Base,
				Quote:        cp.Quote,
				SpotDetails: &config.SpotDetails{
					InitialQuoteFunds: &leet,
				},
				BuySide:  config.MinMax{},
				SellSide: config.MinMax{},
				MakerFee: &decimal.Zero,
				TakerFee: &decimal.Zero,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin,
			DatabaseData: &config.DatabaseData{
				Config: database.Config{
					Enabled: true,
					Driver:  "sqlite3",
					ConnectionDetails: drivers.ConnectionDetails{
						Database: "gocryptotrader.db",
					},
				},
				StartDate:        time.Now().Add(-time.Minute),
				EndDate:          time.Now(),
				InclusiveEndDate: true,
			}},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]interface{}{
				"hello": "moto",
			},
		},
	}
	em := engine.ExchangeManager{}
	exch, err := em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	bt.databaseManager, err = engine.SetupDatabaseConnectionManager(&cfg.DataSettings.DatabaseData.Config)
	if err != nil {
		t.Fatal(err)
	}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	if err != nil && !strings.Contains(err.Error(), "unable to retrieve data from GoCryptoTrader database") {
		t.Error(err)
	}
}

func TestLoadDataCSV(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports: &report.Data{},
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: "Binance",
				Asset:        asset.Spot,
				Base:         cp.Base,
				Quote:        cp.Quote,
				SpotDetails: &config.SpotDetails{
					InitialQuoteFunds: &leet,
				},
				BuySide:  config.MinMax{},
				SellSide: config.MinMax{},
				MakerFee: &decimal.Zero,
				TakerFee: &decimal.Zero,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin,
			CSVData: &config.CSVData{
				FullPath: "test",
			}},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]interface{}{
				"hello": "moto",
			},
		},
	}
	em := engine.ExchangeManager{}
	exch, err := em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	if err != nil &&
		!strings.Contains(err.Error(), "The system cannot find the file specified.") &&
		!strings.Contains(err.Error(), "no such file or directory") {
		t.Error(err)
	}
}

func TestLoadDataLive(t *testing.T) {
	t.Parallel()
	bt := BackTest{
		Reports:  &report.Data{},
		shutdown: make(chan struct{}),
	}
	cp := currency.NewPair(currency.BTC, currency.USDT)
	cfg := &config.Config{
		CurrencySettings: []config.CurrencySettings{
			{
				ExchangeName: "Binance",
				Asset:        asset.Spot,
				Base:         cp.Base,
				Quote:        cp.Quote,
				SpotDetails: &config.SpotDetails{
					InitialQuoteFunds: &leet,
				},
				BuySide:  config.MinMax{},
				SellSide: config.MinMax{},
				MakerFee: &decimal.Zero,
				TakerFee: &decimal.Zero,
			},
		},
		DataSettings: config.DataSettings{
			DataType: common.CandleStr,
			Interval: gctkline.OneMin,
			LiveData: &config.LiveData{
				APIKeyOverride:      "test",
				APISecretOverride:   "test",
				APIClientIDOverride: "test",
				API2FAOverride:      "test",
				RealOrders:          true,
			}},
		StrategySettings: config.StrategySettings{
			Name: dollarcostaverage.Name,
			CustomSettings: map[string]interface{}{
				"hello": "moto",
			},
		},
	}
	em := engine.ExchangeManager{}
	exch, err := em.NewExchangeByName("Binance")
	if err != nil {
		t.Fatal(err)
	}
	exch.SetDefaults()
	b := exch.GetBase()
	b.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	b.CurrencyPairs.Pairs[asset.Spot] = &currency.PairStore{
		Available:     currency.Pairs{cp},
		Enabled:       currency.Pairs{cp},
		AssetEnabled:  convert.BoolPtr(true),
		ConfigFormat:  &currency.PairFormat{Uppercase: true},
		RequestFormat: &currency.PairFormat{Uppercase: true}}
	_, err = bt.loadData(cfg, exch, cp, asset.Spot, false)
	if err != nil {
		t.Error(err)
	}
	bt.Stop()
}
