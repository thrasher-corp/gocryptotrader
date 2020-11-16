package backtest

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	kline2 "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// New returns a new BackTest instance
func New() *BackTest {
	return &BackTest{
		shutdown: make(chan struct{}),
	}
}

// Reset BackTest values to default
func (b *BackTest) Reset() {
	b.EventQueue = nil
	b.Data.Reset()
	b.Portfolio.Reset()
	b.Statistic.Reset()
}

// NewFromConfig takes a strategy config and configures a backtester variable to run
func NewFromConfig(cfg *config.Config) (*BackTest, error) {
	bt := New()
	err := engineBotSetup(cfg)
	if err != nil {
		return nil, err
	}

	exch, fPair, a, err := loadExchangePairAssetBase(cfg)
	if err != nil {
		return nil, err
	}

	bt.Data, err = loadData(cfg, exch, fPair, a)
	if err != nil {
		return nil, err
	}

	var makerFee, takerFee float64
	makerFee, takerFee, err = getFees(exch, fPair)
	if err != nil {
		return nil, err
	}

	bt.Exchange = &exchange.Exchange{
		CurrencySettings: exchange.CurrencySettings{
			CurrencyPair:    fPair,
			AssetType:       a,
			ExchangeFee:     takerFee,
			MakerFee:        takerFee,
			TakerFee:        makerFee,
			MinimumBuySize:  cfg.ExchangeSettings.MinimumBuySize,
			MaximumBuySize:  cfg.ExchangeSettings.MinimumBuySize,
			DefaultBuySize:  cfg.ExchangeSettings.DefaultBuySize,
			MinimumSellSize: cfg.ExchangeSettings.MinimumSellSize,
			MaximumSellSize: cfg.ExchangeSettings.MaximumSellSize,
			DefaultSellSize: cfg.ExchangeSettings.DefaultSellSize,
			CanUseLeverage:  cfg.ExchangeSettings.CanUseLeverage,
			MaximumLeverage: cfg.ExchangeSettings.MaximumLeverage,
		},
		Orders: internalordermanager.Orders{},
	}

	bt.Portfolio = &portfolio.Portfolio{
		InitialFunds: cfg.ExchangeSettings.InitialFunds,
		SizeManager: &size.Size{
			MinimumBuySize:  cfg.PortfolioSettings.MinimumBuySize,
			MaximumBuySize:  cfg.PortfolioSettings.MinimumBuySize,
			DefaultBuySize:  cfg.PortfolioSettings.DefaultBuySize,
			MinimumSellSize: cfg.PortfolioSettings.MinimumSellSize,
			MaximumSellSize: cfg.PortfolioSettings.MaximumSellSize,
			DefaultSellSize: cfg.PortfolioSettings.DefaultSellSize,
			CanUseLeverage:  cfg.PortfolioSettings.CanUseLeverage,
			MaximumLeverage: cfg.PortfolioSettings.MaximumLeverage,
		},
		Funds: cfg.ExchangeSettings.InitialFunds,
		RiskManager: &risk.Risk{
			MaxLeverageRatio:             nil,
			MaxLeverageRate:              nil,
			MaxDiversificationPercentage: nil,
		},
	}

	// TODO: update fee rates after every order to hopefully get new rates
	bt.Portfolio.SetFee(cfg.ExchangeSettings.Name, a, fPair, takerFee)

	bt.Strategy, err = strategies.LoadStrategyByName(cfg.StrategyToLoad)
	if err != nil {
		return nil, err
	}

	bt.Statistic = &statistics.Statistic{
		StrategyName: cfg.StrategyToLoad,
		InitialFunds: cfg.ExchangeSettings.InitialFunds,
	}

	return bt, nil
}

func loadExchangePairAssetBase(cfg *config.Config) (gctexchange.IBotExchange, currency.Pair, asset.Item, error) {
	var err error
	exch := engine.Bot.GetExchangeByName(cfg.ExchangeSettings.Name)
	if exch == nil {
		return nil, currency.Pair{}, "", engine.ErrExchangeNotFound
	}

	var cp, fPair currency.Pair
	cp, err = currency.NewPairFromStrings(cfg.ExchangeSettings.Base, cfg.ExchangeSettings.Quote)
	if err != nil {
		return nil, currency.Pair{}, "", err
	}

	var a asset.Item
	a, err = asset.New(cfg.ExchangeSettings.Asset)
	if err != nil {
		return nil, currency.Pair{}, "", err
	}

	base := exch.GetBase()
	if !base.ValidateAPICredentials() {
		log.Warnf(log.BackTester, "no credentials set for %v, this is theoretical only", base.Name)
	}

	fPair, err = base.FormatExchangeCurrency(cp, a)
	if err != nil {
		return nil, currency.Pair{}, "", err
	}
	return exch, fPair, a, nil
}

func engineBotSetup(cfg *config.Config) error {
	var err error
	engine.Bot, err = engine.NewFromSettings(&engine.Settings{
		EnableDryRun:   true,
		EnableAllPairs: true,
	}, nil)
	if err != nil {
		return err
	}

	err = engine.Bot.LoadExchange(cfg.ExchangeSettings.Name, false, nil)
	if err != nil {
		return err
	}

	err = engine.Bot.OrderManager.Start()
	if err != nil {
		return err
	}

	return nil
}

func getFees(exch gctexchange.IBotExchange, fPair currency.Pair) (makerFee float64, takerFee float64, err error) {
	takerFee, err = exch.GetFeeByType(&gctexchange.FeeBuilder{
		FeeType:       gctexchange.OfflineTradeFee,
		Pair:          fPair,
		IsMaker:       false,
		PurchasePrice: 1,
		Amount:        1,
	})
	if err != nil {
		return makerFee, takerFee, err
	}

	makerFee, err = exch.GetFeeByType(&gctexchange.FeeBuilder{
		FeeType:       gctexchange.OfflineTradeFee,
		Pair:          fPair,
		IsMaker:       true,
		PurchasePrice: 1,
		Amount:        1,
	})
	if err != nil {
		return makerFee, takerFee, err
	}

	return makerFee, takerFee, err
}

func loadData(cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item) (*kline2.DataFromKline, error) {
	base := exch.GetBase()
	if cfg.DatabaseData == nil && cfg.LiveData == nil && cfg.APIData == nil && cfg.CSVData == nil {
		return nil, errors.New("no data settings set in config")
	}
	// load the data
	resp := &kline2.DataFromKline{}
	var err error
	if (cfg.APIData != nil && cfg.DatabaseData != nil) ||
		(cfg.APIData != nil && cfg.LiveData != nil) ||
		(cfg.APIData != nil && cfg.CSVData != nil) ||
		(cfg.DatabaseData != nil && cfg.LiveData != nil) ||
		(cfg.CSVData != nil && cfg.LiveData != nil) ||
		(cfg.CSVData != nil && cfg.DatabaseData != nil) {
		return nil, errors.New("ambiguous settings received. Only one data type can be set")
	}

	if cfg.CSVData != nil {
		switch cfg.CSVData.DataType {
		case candleStr:
			// format: timestamp, volume, open, high, low, close
			csvFile, err := os.Open(cfg.CSVData.FullPath)
			if err != nil {
				return nil, err
			}

			defer func() {
				err = csvFile.Close()
				if err != nil {
					log.Errorln(log.Global, err)
				}
			}()

			csvData := csv.NewReader(csvFile)
			candles := kline.Item{
				Exchange: cfg.ExchangeSettings.Name,
				Pair:     fPair,
				Asset:    a,
				Interval: kline.Interval(cfg.CSVData.Interval),
			}
			for {
				row, errCSV := csvData.Read()
				if errCSV != nil {
					if errCSV == io.EOF {
						break
					}
					return nil, errCSV
				}

				candle := kline.Candle{}
				v, errParse := strconv.ParseInt(row[0], 10, 32)
				if errParse != nil {
					return nil, errParse
				}
				candle.Time = time.Unix(v, 0).UTC()
				if candle.Time.IsZero() {
					err = fmt.Errorf("invalid timestamp received on row %v", row)
					break
				}

				candle.Volume, err = strconv.ParseFloat(row[1], 64)
				if err != nil {
					break
				}

				candle.Open, err = strconv.ParseFloat(row[2], 64)
				if err != nil {
					break
				}

				candle.High, err = strconv.ParseFloat(row[3], 64)
				if err != nil {
					break
				}

				candle.Low, err = strconv.ParseFloat(row[4], 64)
				if err != nil {
					break
				}

				candle.Close, err = strconv.ParseFloat(row[5], 64)
				if err != nil {
					break
				}

				candles.Candles = append(candles.Candles, candle)
			}

			resp.Item = candles
		case tradeStr:
			// format: timestamp, price, amount, side
			csvFile, err := os.Open(cfg.CSVData.FullPath)
			if err != nil {
				return nil, err
			}

			defer func() {
				err = csvFile.Close()
				if err != nil {
					log.Errorln(log.Global, err)
				}
			}()

			csvData := csv.NewReader(csvFile)
			var trades []trade.Data
			for {
				row, errCSV := csvData.Read()
				if errCSV != nil {
					if errCSV == io.EOF {
						break
					}
					return nil, errCSV
				}

				t := trade.Data{}
				v, errParse := strconv.ParseInt(row[0], 10, 32)
				if errParse != nil {
					return nil, errParse
				}
				t.Timestamp = time.Unix(v, 0).UTC()
				if t.Timestamp.IsZero() {
					err = fmt.Errorf("invalid timestamp received on row %v", row)
					break
				}

				t.Price, err = strconv.ParseFloat(row[1], 64)
				if err != nil {
					break
				}

				t.Amount, err = strconv.ParseFloat(row[2], 64)
				if err != nil {
					break
				}

				t.Side, err = order.StringToOrderSide(row[3])
				if err != nil {
					return nil, err
				}

				trades = append(trades, t)
			}
			resp.Item, err = trade.ConvertTradesToCandles(kline.Interval(cfg.CSVData.Interval), trades...)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unrecognised csv datatype received: '%v'", cfg.CSVData.DataType)
		}
	} else if cfg.APIData != nil {
		var candles kline.Item
		switch cfg.APIData.DataType {
		case candleStr:
			candles, err = exch.GetHistoricCandlesExtended(fPair, a, cfg.APIData.StartDate, cfg.APIData.EndDate, kline.Interval(cfg.APIData.Interval))
			if err != nil {
				return nil, err
			}
		case tradeStr:
			var trades []trade.Data
			trades, err = exch.GetHistoricTrades(fPair, a, cfg.APIData.StartDate, cfg.APIData.EndDate)
			if err != nil {
				return nil, err
			}

			candles, err = trade.ConvertTradesToCandles(kline.Interval(cfg.APIData.Interval), trades...)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unrecognised api datatype received: '%v'", cfg.APIData.DataType)
		}

		resp.Item = candles
	} else if cfg.LiveData != nil {
		if cfg.LiveData.APIKeyOverride != "" {
			base.API.Credentials.Key = cfg.LiveData.APIKeyOverride
		}
		if cfg.LiveData.APISecretOverride != "" {
			base.API.Credentials.Secret = cfg.LiveData.APISecretOverride
		}
		if cfg.LiveData.APIClientIDOverride != "" {
			base.API.Credentials.ClientID = cfg.LiveData.APIClientIDOverride
		}
		if cfg.LiveData.API2FAOverride != "" {
			base.API.Credentials.PEMKey = cfg.LiveData.API2FAOverride
		}
		validated := base.ValidateAPICredentials()
		base.API.AuthenticatedSupport = validated
		if !validated {
			log.Warn(log.BackTester, "bad credentials received, no live trading for you")
			cfg.LiveData.RealOrders = false
		}
		go func() {
			candles, err = exch.GetHistoricCandles(fPair, a, time.Now().Add(-cfg.LiveData.Interval), time.Now(), kline.Interval(cfg.LiveData.Interval))
			if err != nil {
				return
			}

		}()
	} else if cfg.DatabaseData != nil {
		if cfg.DatabaseData.ConfigOverride != nil {
			engine.Bot.Config.Database = *cfg.DatabaseData.ConfigOverride
			err = engine.Bot.DatabaseManager.Start(engine.Bot)
			if err != nil {
				return nil, err
			}
		}
		defer func() {
			err = engine.Bot.DatabaseManager.Stop()
			if err != nil {
				log.Error(log.BackTester, err)
			}
		}()
		switch cfg.DatabaseData.DataType {
		case candleStr:
			datarino, err := getCandleDatabaseData(cfg, fPair, a)
			if err != nil {
				return nil, err
			}
			resp.Item = datarino
		case tradeStr:
			trades, err := trade.GetTradesInRange(
				cfg.ExchangeSettings.Name,
				cfg.ExchangeSettings.Asset,
				cfg.ExchangeSettings.Base,
				cfg.ExchangeSettings.Quote,
				cfg.DatabaseData.StartDate,
				cfg.DatabaseData.EndDate)
			if err != nil {
				return nil, err
			}
			datarino, err := trade.ConvertTradesToCandles(
				kline.Interval(cfg.DatabaseData.Interval),
				trades...)
			if err != nil {
				return nil, err
			}
			resp.Item = datarino
		default:
			return nil, fmt.Errorf("unexpected database datatype: '%v'", cfg.DatabaseData.DataType)
		}
	}

	err = resp.Load()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func getCandleDatabaseData(cfg *config.Config, fPair currency.Pair, a asset.Item) (kline.Item, error) {
	datarino, err := kline.LoadFromDatabase(
		cfg.ExchangeSettings.Name,
		fPair,
		a,
		kline.Interval(cfg.DatabaseData.Interval),
		cfg.DatabaseData.StartDate,
		cfg.DatabaseData.EndDate)
	if err != nil {
		return kline.Item{}, err
	}
	return datarino, nil
}

func (b *BackTest) Stop() {
	b.shutdown <- struct{}{}
}

// Run will iterate over loaded data events
// save them and then handle the event based on its type
func (b *BackTest) Run() error {
	for event, ok := b.nextEvent(); true; event, ok = b.nextEvent() {
		if !ok {
			data, ok := b.Data.Next()
			if !ok {
				break
			}
			b.EventQueue = append(b.EventQueue, data)
			continue
		}

		err := b.handleEvent(event)
		if err != nil {
			return err
		}
		b.Statistic.TrackEvent(event)
	}

	return nil
}

func (b *BackTest) RunLive() error {
	timerino := time.NewTimer(time.Minute * 5)
	for {
		select {
		case <-b.shutdown:
			return nil
		case <-timerino.C:
			return errors.New("no data returned in 5 minutes, shutting down")
		default:
			//
			// Go get latest candleStr of interval X, verify that it hasn't been run before, then append the event
			//
			doneARun := false
			for event, ok := b.nextEvent(); true; event, ok = b.nextEvent() {
				doneARun = true
				if !ok {
					data, ok := b.Data.Next()
					if !ok {
						break
					}
					b.EventQueue = append(b.EventQueue, data)
					continue
				}

				err := b.handleEvent(event)
				if err != nil {
					return err
				}
				b.Statistic.TrackEvent(event)
			}
			if doneARun {
				timerino = time.NewTimer(time.Minute * 5)
			}
		}
	}
}

func (b *BackTest) nextEvent() (e interfaces.EventHandler, ok bool) {
	if len(b.EventQueue) == 0 {
		return e, false
	}

	e = b.EventQueue[0]
	b.EventQueue = b.EventQueue[1:]

	return e, true
}

// handleEvent switches based on the eventHandler type
// it will then act on the event and if needed, will add more events to the queue to be handled
func (b *BackTest) handleEvent(e interfaces.EventHandler) error {
	switch event := e.(type) {
	case interfaces.DataEventHandler:
		b.Portfolio.Update(event)
		b.Statistic.Update(event, b.Portfolio)
		s, err := b.Strategy.OnSignal(b.Data, b.Portfolio)
		if err != nil {
			log.Errorf(log.BackTester, "%s - %s", e.GetTime().Format(common.SimpleTimeFormat), err.Error())
			break
		}
		b.EventQueue = append(b.EventQueue, s)

	case signal.SignalEvent:
		cs := b.Exchange.GetCurrency()
		o, err := b.Portfolio.OnSignal(event, b.Data, &cs)
		if err != nil {
			if errors.Is(err, portfolio.NoHoldingsToSellErr) || errors.Is(err, portfolio.NotEnoughFundsErr) {
				log.Warnf(log.BackTester, "%s - %s", e.GetTime().Format(common.SimpleTimeFormat), err.Error())
			} else {
				log.Errorf(log.BackTester, "%s - %s", e.GetTime().Format(common.SimpleTimeFormat), err.Error())
			}
			break
		}
		b.EventQueue = append(b.EventQueue, o)

	case internalordermanager.OrderEvent:
		f, err := b.Exchange.ExecuteOrder(event, b.Data)
		if err != nil {
			log.Errorf(log.BackTester, "%s - %s", e.GetTime().Format(common.SimpleTimeFormat), err.Error())
			break
		}
		b.EventQueue = append(b.EventQueue, f)
	case fill.FillEvent:
		t, err := b.Portfolio.OnFill(event, b.Data)
		if err != nil {
			log.Errorf(log.BackTester, "%s - %s", e.GetTime().Format(common.SimpleTimeFormat), err.Error())
			break
		}
		b.Statistic.TrackTransaction(t)
	}

	return nil
}
