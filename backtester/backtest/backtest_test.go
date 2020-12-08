package backtest

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/goose"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	common2 "github.com/thrasher-corp/gocryptotrader/common"
	config2 "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	dbsqlite3 "github.com/thrasher-corp/gocryptotrader/database/drivers/sqlite3"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	exchange2 "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

const (
	databaseFolder = "database"
	databaseName   = "backtester.db"
)

func genOHCLVData() kline.Item {
	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	var outItem kline.Item
	outItem.Interval = kline.OneDay
	outItem.Asset = asset.Spot
	outItem.Pair = currency.NewPair(currency.BTC, currency.USDT)
	outItem.Exchange = "test"

	outItem.Candles = make([]kline.Candle, 365)
	outItem.Candles[0] = kline.Candle{
		Time:   start,
		Open:   0,
		High:   10 + rand.Float64(),
		Low:    10 + rand.Float64(),
		Close:  10 + rand.Float64(),
		Volume: 10,
	}

	for x := 1; x < 365; x++ {
		outItem.Candles[x] = kline.Candle{
			Time:   start.Add(time.Hour * 24 * time.Duration(x)),
			Open:   outItem.Candles[x-1].Close,
			High:   outItem.Candles[x-1].Open + rand.Float64(),
			Low:    outItem.Candles[x-1].Open - rand.Float64(),
			Close:  outItem.Candles[x-1].Open + rand.Float64(),
			Volume: float64(rand.Int63n(150)),
		}
	}
	outItem.Exchange = "binance"
	return outItem
}

func TestLoadCandleDataFromAPI(t *testing.T) {
	cfg := config.Config{
		StrategyToLoad: "dollarcostaverage",
		CurrencySettings: config.ExchangeSettings{
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
		APIData: &config.APIData{
			StartDate: time.Now().Add(-time.Hour * 24 * 7),
			EndDate:   time.Now(),
			Interval:  kline.OneDay.Duration(),
			DataType:  candleStr,
		},
		PortfolioSettings: config.PortfolioSettings{
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
	bt, err := NewFromConfig(&cfg)
	if err != nil {
		t.Error(err)
	}
}

func TestWhatHAppensWhenLiveIsRun(t *testing.T) {
	cfg := config.Config{
		StrategyToLoad: "dollarcostaverage",
		CurrencySettings: config.ExchangeSettings{
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
		LiveData: &config.LiveData{
			Interval:   kline.OneMin.Duration(),
			RealOrders: false,
		},
		PortfolioSettings: config.PortfolioSettings{
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
	bt, err := NewFromConfig(&cfg)
	if err != nil {
		t.Error(err)
	}
	go func() {
		err = bt.RunLive()
		if err != nil {
			fmt.Print(err)
			os.Exit(-1)
		}
	}()
}

func TestLoadTradeDataFromAPI(t *testing.T) {
	cfg := config.Config{
		StrategyToLoad: "dollarcostaverage",
		CurrencySettings: config.ExchangeSettings{
			Name:            "ftx",
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
		APIData: &config.APIData{
			StartDate: time.Now().Add(-time.Hour * 24 * 7),
			EndDate:   time.Now(),
			Interval:  kline.OneDay.Duration(),
			DataType:  tradeStr,
		},
		PortfolioSettings: config.PortfolioSettings{
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
	bt, err := NewFromConfig(&cfg)
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataFromCandleDatabase(t *testing.T) {
	klineData := genOHCLVData()

	cfg := config.Config{
		StrategyToLoad: "dollarcostaverage",
		CurrencySettings: config.ExchangeSettings{
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
		DatabaseData: &config.DatabaseData{
			DataType:  candleStr,
			StartDate: klineData.Candles[0].Time,
			EndDate:   klineData.Candles[len(klineData.Candles)-1].Time,
			Interval:  kline.OneDay.Duration(),
			ConfigOverride: &database.Config{
				Enabled: true,
				Driver:  "sqlite",
				ConnectionDetails: drivers.ConnectionDetails{
					Host:     "localhost",
					Database: databaseName,
				},
			},
		},
		PortfolioSettings: config.PortfolioSettings{
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

	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		t.Error(err)
		return
	}
	engine.Bot.Config = &config2.Config{}
	err = engine.Bot.Config.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Errorf("SetupTest: Failed to load config: %s", err)
		return
	}
	engine.Bot.Config.Database = *cfg.DatabaseData.ConfigOverride
	database.DB.Config = cfg.DatabaseData.ConfigOverride
	if engine.Bot.GetExchangeByName("binance") == nil {
		err = engine.Bot.LoadExchange("binance", false, nil)
		if err != nil {
			t.Errorf("SetupTest: Failed to load exchange: %s", err)
			return
		}
	}

	dbConn, err := dbsqlite3.Connect()
	if err != nil {
		t.Errorf("database failed to connect: %v, some features that utilise a database will be unavailable", err)
	}
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {

	}()
	path := filepath.Join("..", "..", "database", "migrations")
	err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		t.Errorf("failed to run migrations %v", err)
		return
	}

	uuider, _ := uuid.NewV4()
	err = exchange2.Insert(exchange2.Details{Name: "binance", UUID: uuider})
	if err != nil {
		t.Errorf("failed to insert exchange %v", err)
		return
	}
	binanceExchange := engine.Bot.GetExchangeByName("binance")
	_, err = kline.StoreInDatabase(&klineData, false)
	if err != nil {
		t.Error(err)
		return
	}
	err = dbConn.SQL.Close()
	if err != nil {
		t.Error(err)
	}

	bt, err := loadData(
		&cfg,
		binanceExchange,
		currency.NewPair(currency.BTC, currency.USDT),
		asset.Spot)

	if err != nil {
		t.Error(err)
		return
	}
	if len(bt.Data.List()) == 0 {
		t.Error("no data loaded")
	}
	err = os.Remove(filepath.Join(common2.GetDefaultDataDir(runtime.GOOS), databaseFolder, databaseName))
	if err != nil {
		t.Error(err)
	}
}

func TestLoadDataFromTradeDatabase(t *testing.T) {
	cfg := config.Config{
		StrategyToLoad: "dollarcostaverage",
		CurrencySettings: config.ExchangeSettings{
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
		DatabaseData: &config.DatabaseData{
			DataType:  tradeStr,
			StartDate: time.Now().Add(-time.Hour * 100),
			EndDate:   time.Now(),
			Interval:  kline.OneHour.Duration(),
			ConfigOverride: &database.Config{
				Enabled: true,
				Driver:  "sqlite",
				ConnectionDetails: drivers.ConnectionDetails{
					Host:     "localhost",
					Database: databaseName,
				},
			},
		},
		PortfolioSettings: config.PortfolioSettings{
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

	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		t.Error(err)
		return
	}
	engine.Bot.Config = &config2.Config{}
	err = engine.Bot.Config.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Errorf("SetupTest: Failed to load config: %s", err)
		return
	}
	engine.Bot.Config.Database = *cfg.DatabaseData.ConfigOverride
	database.DB.Config = cfg.DatabaseData.ConfigOverride
	if engine.Bot.GetExchangeByName("binance") == nil {
		err = engine.Bot.LoadExchange("binance", false, nil)
		if err != nil {
			t.Errorf("SetupTest: Failed to load exchange: %s", err)
			return
		}
	}

	dbConn, err := dbsqlite3.Connect()
	if err != nil {
		t.Errorf("database failed to connect: %v, some features that utilise a database will be unavailable", err)
	}
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {

	}()
	path := filepath.Join("..", "..", "database", "migrations")
	err = goose.Run("up", dbConn.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		t.Errorf("failed to run migrations %v", err)
		return
	}

	uuider, _ := uuid.NewV4()
	err = exchange2.Insert(exchange2.Details{Name: "binance", UUID: uuider})
	if err != nil {
		t.Errorf("failed to insert exchange %v", err)
		return
	}
	binanceExchange := engine.Bot.GetExchangeByName("binance")

	var trades []trade.Data
	for i := int64(0); i < 100; i++ {
		trades = append(trades, trade.Data{
			TID:          strconv.FormatInt(i, 10),
			Exchange:     "binance",
			CurrencyPair: currency.NewPair(currency.BTC, currency.USDT),
			AssetType:    asset.Spot,
			Side:         order.Buy,
			Price:        float64(i) * 1.337,
			Amount:       float64(i) * 1.337,
			Timestamp:    time.Now().Add(-time.Hour * time.Duration(i)),
		})
	}

	err = trade.SaveTradesToDatabase(trades...)
	if err != nil {
		t.Error(err)
		return
	}
	err = dbConn.SQL.Close()
	if err != nil {
		t.Error(err)
	}

	bt, err := loadData(
		&cfg,
		binanceExchange,
		currency.NewPair(currency.BTC, currency.USDT),
		asset.Spot)

	if err != nil {
		t.Error(err)
		return
	}
	if len(bt.Data.List()) == 0 {
		t.Error("no data loaded")
	}
	err = os.Remove(filepath.Join(common2.GetDefaultDataDir(runtime.GOOS), databaseFolder, databaseName))
	if err != nil {
		t.Error(err)
	}
}

func TestLoadCandleDataFromCSV(t *testing.T) {
	cfg := config.Config{
		StrategyToLoad: "dollarcostaverage",
		CurrencySettings: config.ExchangeSettings{
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
		CSVData: &config.CSVData{
			DataType: candleStr,
			Interval: kline.OneHour.Duration(),
			FullPath: filepath.Join("..", "..", "testdata", "binance_BTCUSDT_24h_2019_01_01_2020_01_01.csv"),
		},
		PortfolioSettings: config.PortfolioSettings{
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

	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		t.Error(err)
		return
	}
	engine.Bot.Config = &config2.Config{}
	err = engine.Bot.Config.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Errorf("SetupTest: Failed to load config: %s", err)
		return
	}
	if engine.Bot.GetExchangeByName("binance") == nil {
		err = engine.Bot.LoadExchange("binance", false, nil)
		if err != nil {
			t.Errorf("SetupTest: Failed to load exchange: %s", err)
			return
		}
	}
	binanceExchange := engine.Bot.GetExchangeByName("binance")

	bt, err := loadData(
		&cfg,
		binanceExchange,
		currency.NewPair(currency.BTC, currency.USDT),
		asset.Spot)

	if err != nil {
		t.Error(err)
		return
	}
	if len(bt.Data.List()) == 0 {
		t.Error("no data loaded")
	}
}

func TestLoadTradeDataFromCSV(t *testing.T) {
	cfg := config.Config{
		StrategyToLoad: "dollarcostaverage",
		CurrencySettings: config.ExchangeSettings{
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
		CSVData: &config.CSVData{
			DataType: tradeStr,
			Interval: kline.OneMin.Duration(),
			FullPath: filepath.Join("..", "..", "testdata", "binance_BTCUSDT_24h-trades_2020_11_16.csv"),
		},
		PortfolioSettings: config.PortfolioSettings{
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

	var err error
	engine.Bot, err = engine.New()
	if err != nil {
		t.Error(err)
		return
	}
	engine.Bot.Config = &config2.Config{}
	err = engine.Bot.Config.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Errorf("SetupTest: Failed to load config: %s", err)
		return
	}
	if engine.Bot.GetExchangeByName("binance") == nil {
		err = engine.Bot.LoadExchange("binance", false, nil)
		if err != nil {
			t.Errorf("SetupTest: Failed to load exchange: %s", err)
			return
		}
	}
	binanceExchange := engine.Bot.GetExchangeByName("binance")

	bt, err := loadData(
		&cfg,
		binanceExchange,
		currency.NewPair(currency.BTC, currency.USDT),
		asset.Spot)

	if err != nil {
		t.Error(err)
		return
	}
	if len(bt.Data.List()) == 0 {
		t.Error("no data loaded")
	}
}
