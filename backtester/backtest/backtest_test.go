package backtest

import (
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/goose"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	kline2 "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	config2 "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/database/drivers"
	"github.com/thrasher-corp/gocryptotrader/database/repository"
	exchange2 "github.com/thrasher-corp/gocryptotrader/database/repository/exchange"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type TestStrategy struct{}

func (s *TestStrategy) Name() string {
	return "TestStrategy"
}

func (s *TestStrategy) OnSignal(d interfaces.DataHandler, p portfolio.PortfolioHandler) (signal.SignalEvent, error) {
	signal := signal.Signal{
		Event: event.Event{Time: d.Latest().GetTime(),
			CurrencyPair: d.Latest().Pair()},
	}
	log.Printf("STREAM CLOSE at: %v", d.StreamClose())

	rsi := indicators.RSI(d.StreamClose(), 14)
	latestRSI := rsi[len(rsi)-1]
	log.Printf("RSI at: %v", latestRSI)
	if latestRSI <= 30 {
		// oversold, time to buy like a sweet pro
		signal.Direction = order.Buy
	} else if latestRSI >= 70 {
		// overbought, time to sell because granny is talking about BTC again
		signal.Direction = order.Sell
	} else {
		signal.Direction = common.DoNothing
	}

	return &signal, nil
}

func TestBackTest(t *testing.T) {
	bt := New()

	data := &kline2.DataFromKline{
		Item: genOHCLVData(),
	}
	err := data.Load()
	if err != nil {
		t.Fatal(err)
	}

	bt.Data = data
	bt.Portfolio = &portfolio.Portfolio{
		InitialFunds: 1337,
		SizeManager:  &size.Size{},
		RiskManager:  &risk.Risk{},
	}

	bt.Strategy = &TestStrategy{}
	bt.Exchange = &exchange.Exchange{
		CurrencySettings: exchange.CurrencySettings{
			CurrencyPair: currency.Pair{
				Delimiter: "-",
				Base:      currency.BTC,
				Quote:     currency.USD,
			},
			AssetType: asset.Spot,
		},
	}

	statistic := statistics.Statistic{
		StrategyName: "HelloWorld",
	}
	bt.Statistic = &statistic
	err = bt.Run()
	if err != nil {
		t.Fatal(err)
	}
	ret := statistic.ReturnResults()
	for x := range ret.Transactions {
		fmt.Println(ret.Transactions[x])
	}
	fmt.Printf("Total Events: %v | Total Transactions: %v\n", ret.TotalEvents, ret.TotalTransactions)

	bt.Reset()
}

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

	return outItem
}

func TestLoadDataFromAPI(t *testing.T) {
	cfg := config.Config{
		StrategyToLoad: "dollarcostaverage",
		ExchangeSettings: config.ExchangeSettings{
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
		CandleData: &config.CandleData{
			StartDate: time.Now().Add(-time.Hour * 24 * 7),
			EndDate:   time.Now(),
			Interval:  kline.OneHour.Duration(),
		},
		DatabaseData: nil,
		LiveData:     nil,
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
	if len(bt.Data.List()) == 0 {
		t.Error("no data loaded")
	}
}

func TestLoadDataFromCandleDatabase(t *testing.T) {
	cfg := config.Config{
		StrategyToLoad: "dollarcostaverage",
		ExchangeSettings: config.ExchangeSettings{
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
			DataType:  "candle",
			StartDate: time.Now().Add(-time.Hour),
			EndDate:   time.Now(),
			Interval:  kline.OneMin.Duration(),
			ConfigOverride: &database.Config{
				Enabled: true,
				Verbose: false,
				Driver:  "sqlite",
				ConnectionDetails: drivers.ConnectionDetails{
					Host:     "localhost",
					Database: "superbutts",
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
		t.Fatal(err)
	}
	engine.Bot.Config = &config2.Config{}
	err = engine.Bot.Config.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		t.Fatalf("SetupTest: Failed to load config: %s", err)
	}

	if engine.Bot.GetExchangeByName("Bitstamp") == nil {
		err = engine.Bot.LoadExchange("Bitstamp", false, nil)
		if err != nil {
			t.Fatalf("SetupTest: Failed to load exchange: %s", err)
		}
	}
	err = engine.Bot.DatabaseManager.Start(engine.Bot)
	what := &database.Instance{
		SQL:       nil,
		DataPath:  "",
		Config:    cfg.DatabaseData.ConfigOverride,
		Connected: false,
		Mu:        sync.RWMutex{},
	}

	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join("..", "..", "database", "migrations")
	err = goose.Run("up", what.SQL, repository.GetSQLDialect(), path, "")
	if err != nil {
		t.Fatalf("failed to run migrations %v", err)
	}
	uuider, _ := uuid.NewV4()
	err = exchange2.Insert(exchange2.Details{Name: "Bitstamp", UUID: uuider})
	if err != nil {
		t.Fatalf("failed to insert exchange %v", err)
	}
	bt, err := loadData(
		&cfg,
		engine.Bot.GetExchangeByName("Bitstamp"),
		currency.NewPair(currency.BTC, currency.USD),
		asset.Spot)

	if err != nil {
		t.Error(err)
	}
	if len(bt.Data.List()) == 0 {
		t.Error("no data loaded")
	}
}
