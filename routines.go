package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	"github.com/thrasher-/gocryptotrader/currency/symbol"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
	"github.com/thrasher-/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-/gocryptotrader/exchanges/stats"
	"github.com/thrasher-/gocryptotrader/exchanges/ticker"
)

const (
	updateTicker    = "update_ticker"
	updateOrderbook = "update_orderbook"
	updateHistory   = "update_history"

	maxWorkers   = 5
	maxFrameSize = 250 * time.Millisecond
)

func printCurrencyFormat(price float64) string {
	displaySymbol, err := symbol.GetSymbolByCurrencyName(bot.config.Currency.FiatDisplayCurrency)
	if err != nil {
		log.Printf("Failed to get display symbol: %s", err)
	}

	return fmt.Sprintf("%s%.8f", displaySymbol, price)
}

func printConvertCurrencyFormat(origCurrency string, origPrice float64) string {
	displayCurrency := bot.config.Currency.FiatDisplayCurrency
	conv, err := currency.ConvertCurrency(origPrice, origCurrency, displayCurrency)
	if err != nil {
		log.Printf("Failed to convert currency: %s", err)
	}

	displaySymbol, err := symbol.GetSymbolByCurrencyName(displayCurrency)
	if err != nil {
		log.Printf("Failed to get display symbol: %s", err)
	}

	origSymbol, err := symbol.GetSymbolByCurrencyName(origCurrency)
	if err != nil {
		log.Printf("Failed to get original currency symbol: %s", err)
	}

	return fmt.Sprintf("%s%.2f %s (%s%.2f %s)",
		displaySymbol,
		conv,
		displayCurrency,
		origSymbol,
		origPrice,
		origCurrency,
	)
}

func printTickerSummary(result ticker.Price, p pair.CurrencyPair, assetType, exchangeName string, err error) {
	if err != nil {
		log.Printf("Failed to get %s %s ticker. Error: %s",
			p.Pair().String(),
			exchangeName,
			err)
		return
	}

	stats.Add(exchangeName, p, assetType, result.Last, result.Volume)
	if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.String() != bot.config.Currency.FiatDisplayCurrency {
		origCurrency := p.SecondCurrency.Upper().String()
		log.Printf("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
			exchangeName,
			exchange.FormatCurrency(p).String(),
			assetType,
			printConvertCurrencyFormat(origCurrency, result.Last),
			printConvertCurrencyFormat(origCurrency, result.Ask),
			printConvertCurrencyFormat(origCurrency, result.Bid),
			printConvertCurrencyFormat(origCurrency, result.High),
			printConvertCurrencyFormat(origCurrency, result.Low),
			result.Volume)
	} else {
		if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.Upper().String() == bot.config.Currency.FiatDisplayCurrency {
			log.Printf("%s %s %s: TICKER: Last %s Ask %s Bid %s High %s Low %s Volume %.8f",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				printCurrencyFormat(result.Last),
				printCurrencyFormat(result.Ask),
				printCurrencyFormat(result.Bid),
				printCurrencyFormat(result.High),
				printCurrencyFormat(result.Low),
				result.Volume)
		} else {
			log.Printf("%s %s %s: TICKER: Last %.8f Ask %.8f Bid %.8f High %.8f Low %.8f Volume %.8f",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				result.Last,
				result.Ask,
				result.Bid,
				result.High,
				result.Low,
				result.Volume)
		}
	}
}

func printOrderbookSummary(result orderbook.Base, p pair.CurrencyPair, assetType, exchangeName string, err error) {
	if err != nil {
		log.Printf("Failed to get %s %s orderbook. Error: %s",
			p.Pair().String(),
			exchangeName,
			err)
		return
	}

	bidsAmount, bidsValue := result.CalculateTotalBids()
	asksAmount, asksValue := result.CalculateTotalAsks()

	if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.String() != bot.config.Currency.FiatDisplayCurrency {
		origCurrency := p.SecondCurrency.Upper().String()
		log.Printf("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
			exchangeName,
			exchange.FormatCurrency(p).String(),
			assetType,
			len(result.Bids),
			bidsAmount,
			p.FirstCurrency.String(),
			printConvertCurrencyFormat(origCurrency, bidsValue),
			len(result.Asks),
			asksAmount,
			p.FirstCurrency.String(),
			printConvertCurrencyFormat(origCurrency, asksValue),
		)
	} else {
		if currency.IsFiatCurrency(p.SecondCurrency.String()) && p.SecondCurrency.Upper().String() == bot.config.Currency.FiatDisplayCurrency {
			log.Printf("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %s Asks len: %d Amount: %f %s. Total value: %s",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				len(result.Bids),
				bidsAmount,
				p.FirstCurrency.String(),
				printCurrencyFormat(bidsValue),
				len(result.Asks),
				asksAmount,
				p.FirstCurrency.String(),
				printCurrencyFormat(asksValue),
			)
		} else {
			log.Printf("%s %s %s: ORDERBOOK: Bids len: %d Amount: %f %s. Total value: %f Asks len: %d Amount: %f %s. Total value: %f",
				exchangeName,
				exchange.FormatCurrency(p).String(),
				assetType,
				len(result.Bids),
				bidsAmount,
				p.FirstCurrency.String(),
				bidsValue,
				len(result.Asks),
				asksAmount,
				p.FirstCurrency.String(),
				asksValue,
			)
		}
	}
}

func relayWebsocketEvent(result interface{}, event, assetType, exchangeName string) {
	evt := WebsocketEvent{
		Data:      result,
		Event:     event,
		AssetType: assetType,
		Exchange:  exchangeName,
	}
	err := BroadcastWebsocketMessage(evt)
	if err != nil {
		log.Println(fmt.Errorf("Failed to broadcast websocket event. Error: %s",
			err))
	}
}

// Update is the main update type monitoring multiple assets
var update Monitor

// Monitor defines the Enabled and disabled trading assets
type Monitor struct {
	// Enabled trading assets
	Enabled map[string][]*TradingAsset

	// Disabled trading assets, so we dont iterate over non-functional assets
	// but keep them in memory if enabled later on
	Disabled []*TradingAsset

	shutdown        chan struct{}      // main shutdown
	refreshExchange chan string        // batched timer shutdown
	doWork          chan WorkerRequest // to worker pool
	toFramer        chan WorkerRequest // to framing routine
	finisher        chan WorkerRequest // to finisher routine

	wg           sync.WaitGroup             // main wait group
	jobWg        map[string]*sync.WaitGroup // job specific wait group for an exchange for job creation and tracking
	WebsocketMtx sync.Mutex                 // websocket mutex
	sync.Mutex
}

// TradingAsset defines a supported trading asset on an exchange
type TradingAsset struct {
	Exchange     exchange.IBotExchange
	AssetType    string
	CurrencyPair pair.CurrencyPair

	Ticker            TradeData
	Orderbook         TradeData
	History           TradeData
	WebsocketOverride bool
}

// TradeData defines actually request trade data
type TradeData struct {
	Enabled     bool
	LastUpdated time.Time
}

// StartMonitor starts a full monitor for all enabled cryptocurrency asset pairs
func StartMonitor(exchanges []exchange.IBotExchange, ticker, orderbook, history, verbose bool) error {
	log.Println("GoCryptoTrader - asset monitor service started")

	update = Monitor{
		Enabled:  make(map[string][]*TradingAsset),
		shutdown: make(chan struct{}, 1),
		doWork:   make(chan WorkerRequest, 1000),
		toFramer: make(chan WorkerRequest, 1000),
		finisher: make(chan WorkerRequest, 1000),
		jobWg:    make(map[string]*sync.WaitGroup),
	}

	for i := range exchanges {
		availableCurrencies := exchanges[i].GetAvailableCurrencies()
		enabledCurrencies := exchanges[i].GetEnabledCurrencies()

		// TODO - Update asset retrieval

		for _, availableCurrency := range availableCurrencies {
			var isEnabled bool
			for _, enabledCurrency := range enabledCurrencies {
				if availableCurrency == enabledCurrency {
					isEnabled = true
					break
				}
			}

			tradingAsset := &TradingAsset{
				Exchange:     exchanges[i],
				AssetType:    "SPOT",
				CurrencyPair: availableCurrency,
				Ticker:       TradeData{Enabled: ticker},
				Orderbook:    TradeData{Enabled: orderbook},
				History:      TradeData{Enabled: history},
			}

			if isEnabled {
				update.Enabled[common.StringToUpper(exchanges[i].GetName())] = append(update.Enabled[common.StringToUpper(exchanges[i].GetName())], tradingAsset)
				continue
			}

			update.Disabled = append(update.Disabled, tradingAsset)
		}
	}

	go update.WorkFramer(verbose)
	go update.Finalise(verbose)

	// Creates workers in a pool
	for i := 0; i < maxWorkers; i++ {
		go update.Worker(i, verbose)
	}

	update.StartJobs()

	return nil
}

// UpdateTicker updates ticker details in the asset register
func (m *Monitor) UpdateTicker(t exchange.TickerData) error {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	for i := range m.Enabled[common.StringToUpper(t.Exchange)] {
		if common.StringToUpper(m.Enabled[common.StringToUpper(t.Exchange)][i].AssetType) == common.StringToUpper(t.AssetType) &&
			m.Enabled[common.StringToUpper(t.Exchange)][i].CurrencyPair.Display("", true) == t.Pair.Display("", true) {
			m.Enabled[common.StringToUpper(t.Exchange)][i].Ticker.LastUpdated = time.Now()
			return nil
		}
	}

	return fmt.Errorf("routines.go error - can not find ticker trading asset for Exchange:%s, Asset:%s, CurrencyPair:%s",
		t.Exchange,
		t.AssetType,
		t.Pair.Display("", true))
}

// UpdateOrderbook updates orderbook details in the asset register
func (m *Monitor) UpdateOrderbook(o orderbook.Base, exchangeName string) error {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	for i := range m.Enabled[common.StringToUpper(exchangeName)] {
		if common.StringToUpper(m.Enabled[common.StringToUpper(exchangeName)][i].AssetType) == common.StringToUpper(o.AssetType) &&
			m.Enabled[common.StringToUpper(exchangeName)][i].CurrencyPair.Display("", true) == o.Pair.Display("", true) {
			m.Enabled[common.StringToUpper(exchangeName)][i].Ticker.LastUpdated = time.Now()
			return nil
		}
	}
	return fmt.Errorf("routines.go error - can not find orderbook trading asset for Exchange:%s, Asset:%s, CurrencyPair:%s",
		exchangeName,
		o.AssetType,
		o.Pair.Display("", true))
}

// SwitchRESTFunctionality stops REST requests when websocket feeds are active
// and restarts REST when connection fails
func (m *Monitor) SwitchRESTFunctionality(websocketActive bool, exchangeName string) {
	m.WebsocketMtx.Lock()
	if websocketActive {
		for x := range m.Enabled {
			for y := range m.Enabled[x] {
				if m.Enabled[x][y].Exchange.GetName() == exchangeName {
					m.Enabled[x][y].WebsocketOverride = true
				}
			}
		}

		rMtx.Lock()
		for x := range register {
			for y := range register[x] {
				register[x][y] <- struct{}{}
			}
		}
		rMtx.Unlock()

		m.jobWg[exchangeName].Wait()
	} else {
		m.StartJobs()
	}
	m.WebsocketMtx.Lock()
}

// StartJobs creates tickets of jobs that are enabled per exchange, asset type,
// currency pair
func (m *Monitor) StartJobs() {
	for _, exchangeJobs := range m.Enabled {
		for _, job := range exchangeJobs {
			job.WebsocketOverride = false
			if m.jobWg[job.Exchange.GetName()] == nil {
				var wg sync.WaitGroup
				m.jobWg[job.Exchange.GetName()] = &wg
			}

			if job.Orderbook.Enabled {
				m.toFramer <- WorkerRequest{
					Exchange:     job.Exchange,
					Update:       updateOrderbook,
					CurrencyPair: job.CurrencyPair,
					Asset:        job.AssetType,
					Reset:        true,
					Override:     &job.WebsocketOverride,
				}
				m.jobWg[job.Exchange.GetName()].Add(1)
			}

			if job.Ticker.Enabled {
				m.toFramer <- WorkerRequest{
					Exchange:     job.Exchange,
					Update:       updateTicker,
					CurrencyPair: job.CurrencyPair,
					Asset:        job.AssetType,
					Reset:        true,
					Override:     &job.WebsocketOverride,
				}
				m.jobWg[job.Exchange.GetName()].Add(1)
			}

			if job.History.Enabled {
				m.toFramer <- WorkerRequest{
					Exchange:     job.Exchange,
					Update:       updateHistory,
					CurrencyPair: job.CurrencyPair,
					Asset:        job.AssetType,
					Reset:        true,
					Override:     &job.WebsocketOverride,
				}
				m.jobWg[job.Exchange.GetName()].Add(1)
			}
		}
	}
}

// WorkerRequest defines a job to be executed
type WorkerRequest struct {
	Reset        bool
	Exchange     exchange.IBotExchange
	Update       string
	CurrencyPair pair.CurrencyPair
	Asset        string
	LastUpdated  time.Time
	Override     *bool
}

// Finalise updates timer information and cycles jobs
func (m *Monitor) Finalise(verbose bool) {
	m.wg.Add(1)
	defer m.wg.Done()

	for {
		select {
		case <-m.shutdown:
			return

		case job := <-m.finisher:
			go func() {
				m.Lock()
				for i := range m.Enabled[job.Exchange.GetName()] {
					if m.Enabled[job.Exchange.GetName()][i].CurrencyPair == job.CurrencyPair &&
						m.Enabled[job.Exchange.GetName()][i].AssetType == job.Asset {
						switch job.Update {
						case updateTicker:
							m.Enabled[job.Exchange.GetName()][i].Ticker.LastUpdated = job.LastUpdated
							if m.Enabled[job.Exchange.GetName()][i].Ticker.Enabled {
								m.toFramer <- job
							}
							return

						case updateOrderbook:
							m.Enabled[job.Exchange.GetName()][i].Orderbook.LastUpdated = job.LastUpdated
							if m.Enabled[job.Exchange.GetName()][i].Orderbook.Enabled {
								m.toFramer <- job
							}
							return

						case updateHistory:
							m.Enabled[job.Exchange.GetName()][i].History.LastUpdated = job.LastUpdated
							if m.Enabled[job.Exchange.GetName()][i].History.Enabled {
								m.toFramer <- job
							}
							return
						}
					}
				}

				for i := range m.Disabled {
					if m.Disabled[i].CurrencyPair == job.CurrencyPair &&
						m.Disabled[i].AssetType == job.Asset &&
						m.Disabled[i].Exchange.GetName() == job.Exchange.GetName() {
						switch job.Update {
						case updateTicker:
							m.Disabled[i].Ticker.LastUpdated = job.LastUpdated

						case updateOrderbook:
							m.Disabled[i].Orderbook.LastUpdated = job.LastUpdated

						case updateHistory:
							m.Disabled[i].History.LastUpdated = job.LastUpdated

						default:
							log.Fatal("Could not find trading asset at all", job)
						}
					}
				}
				m.Unlock()
			}()
		}
	}
}

// WorkFramer frames jobs in 250ms frames to minimise CPU usage in main check
// routine & minimise multiple routines in time.Sleep
func (m *Monitor) WorkFramer(verbose bool) {
	if verbose {
		log.Println("routines.go WorkFramer() - started")
	}

	m.wg.Add(1)
	defer m.wg.Done()

	ticker := time.NewTicker(maxFrameSize)

	var jobs []WorkerRequest

	for {
		select {
		case <-m.shutdown:
			return

		case job := <-m.toFramer:
			if *job.Override {
				if verbose {
					log.Println("routines.go WorkFramer() websocket override - job dropped")
				}

				m.jobWg[job.Exchange.GetName()].Done()
				continue
			}

			if job.Reset {
				// On initial start and websocket disconnect this will bypass
				// timer dispatch
				job.Reset = false
				m.doWork <- job
				continue
			}

			jobs = append(jobs, job)

		case <-ticker.C:
			if len(jobs) != 0 {
				go m.TimerDispatch(jobs)
				jobs = nil
			}
		}
	}
}

var register = make(map[int][]chan struct{})
var rMtx sync.Mutex
var rWg sync.WaitGroup

// TimerDispatch routine sleeps batched requests via time delay
func (m *Monitor) TimerDispatch(jobs []WorkerRequest) {
	timer := time.NewTimer(10 * time.Second)
	refresh := make(chan struct{}, 1)
	var registeredNumber int

	rMtx.Lock()
	rWg.Add(1)
	registeredNumber = len(register) - 1
	register[registeredNumber] = append(register[registeredNumber], refresh)
	rMtx.Unlock()

	for {
		select {
		case <-refresh:
			// When websocket connects on an exchange this will drop all exchange
			//jobs
			var newList []WorkerRequest
			for _, job := range jobs {
				if *job.Override {
					m.jobWg[job.Exchange.GetName()].Done()
					continue
				}
				newList = append(newList, job)
			}
			jobs = newList
			rWg.Done()
			rMtx.Lock()
			rWg.Add(1)
			rMtx.Unlock()

		case <-timer.C:
			for _, job := range jobs {
				if *job.Override {
					// Secondary catch if webosocket connects - might be redundant
					m.jobWg[job.Exchange.GetName()].Done()
					continue
				}
				m.doWork <- job
			}

			// deregister
			rMtx.Lock()
			delete(register, registeredNumber)
			rWg.Done()
			rMtx.Unlock()
			return
		}
	}
}

// Worker is a worker routine that executes work from the doWork channel
func (m *Monitor) Worker(id int, verbose bool) {
	if verbose {
		log.Printf("routines.go - Worker routine started id: %d", id)
	}

	m.wg.Add(1)
	defer m.wg.Done()

	for {
		select {
		case <-m.shutdown:
			return

		case job := <-m.doWork:
			if verbose {
				log.Printf("worker routine ID: %d job recieved for Exchange: %s CurrencyPair: %s Asset: %s Update: %s",
					id,
					job.Exchange.GetName(),
					job.CurrencyPair.Pair().String(),
					job.Asset,
					job.Update)
			}

			if *job.Override {
				// Edge catch if websocket connects
				m.jobWg[job.Exchange.GetName()].Done()
				continue
			}

			switch job.Update {
			case updateTicker:
				err := m.ProcessTickerREST(job.Exchange, job.CurrencyPair, job.Asset, job.Exchange.GetName(), verbose)
				if err != nil {
					switch err.Error() {
					default:
						if common.StringContains(err.Error(), "connection reset by peer") {
							log.Printf("%s for %s and %s asset type - connection reset by peer retrying....",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else if common.StringContains(err.Error(), "net/http: request canceled") {
							log.Printf("%s for %s and %s asset type - connection request canceled retrying....",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else if common.StringContains(err.Error(), "ValidateData() error") {
							if common.StringContains(err.Error(), "insufficient returned data") {
								log.Printf("%s for %s and %s asset type %s disabling routine",
									job.Exchange.GetName(),
									job.CurrencyPair.Pair().String(),
									job.Asset,
									err.Error())
								// TODO SERIOUS Error! disable job fetching and log
							}
							log.Printf("%s for %s and %s asset type %s continuing",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset,
								err.Error())
							// TODO SERIOUS Error! disable job fetching and log
						} else {
							log.Printf("%s Ticker Updater Routine error %s disabling ticker fetcher routine",
								job.Exchange.GetName(),
								err.Error())
							// TODO disable job fetching and log
						}
					}
				}

			case updateOrderbook:
				err := m.ProcessOrderbookREST(job.Exchange, job.CurrencyPair, job.Asset, verbose)
				if err != nil {
					switch err.Error() {
					default:
						if common.StringContains(err.Error(), "connection reset by peer") {
							log.Printf("%s for %s and %s asset type - connection reset by peer retrying....",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else if common.StringContains(err.Error(), "net/http: request canceled") {
							log.Printf("%s for %s and %s asset type - connection request canceled retrying....",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else if common.StringContains(err.Error(), "ValidateData() error") {
							if common.StringContains(err.Error(), "insufficient returned data") {
								log.Printf("%s for %s and %s asset type %s disabling routine",
									job.Exchange.GetName(),
									job.CurrencyPair.Pair().String(),
									job.Asset,
									err.Error())
								// TODO SERIOUS Error! disable job fetching and log
							}
							log.Printf("%s for %s and %s asset type - %s continuing",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset,
								err.Error())
							// TODO SERIOUS Error! disable job fetching and log
						} else {
							log.Printf("%s Orderbook Updater Routine error %s disabling orderbook fetcher routine",
								job.Exchange.GetName(),
								err.Error())
							// TODO disable job fetching and log
						}
					}
				}

			case updateHistory:
				err := m.ProcessHistoryREST(job.Exchange, job.CurrencyPair, job.Asset, verbose)
				if err != nil {
					switch err.Error() {
					case "history up to date":
						if verbose {
							log.Printf("%s history is up to date for %s as %s asset type, sleeping for 5 mins",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO sleep job 5 minutes
						}

					case "no history returned":
						log.Printf("warning %s no history has been returned for for %s as %s asset type, disabling fetcher routine",
							job.Exchange.GetName(),
							job.CurrencyPair.Pair().String(),
							job.Asset)
					// TODO disable job

					case "trade history not yet implemented":
						log.Printf("%s exchange GetExchangeHistory function not enabled, disabling fetcher routine for %s as %s asset type",
							job.Exchange.GetName(),
							job.CurrencyPair.Pair().String(),
							job.Asset)
						// TODO disable job

					default:
						if common.StringContains(err.Error(), "net/http: request canceled") {
							log.Printf("%s exchange error for %s as %s asset type - net/http: request canceled, retrying",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset)
							// TODO retry every 30 seconds
						} else {
							log.Printf("%s exchange error for %s as %s asset type - %s, disabling fetcher routine",
								job.Exchange.GetName(),
								job.CurrencyPair.Pair().String(),
								job.Asset,
								err.Error())
							// General error end job disable request function
						}
					}
				}
			}
			job.LastUpdated = time.Now() // update time on main asset ledger
			m.toFramer <- job            // re-circulate job
		}
	}
}

// ProcessTickerREST processes tickers data, utilising a REST endpoint
func (m *Monitor) ProcessTickerREST(exch exchange.IBotExchange, p pair.CurrencyPair, assetType, exchangeName string, verbose bool) error {
	result, err := exch.UpdateTicker(p, assetType)
	printTickerSummary(result, p, assetType, exchangeName, err)
	if err != nil {
		return err
	}

	err = m.ValidateData(result)
	if err != nil {
		return err
	}

	bot.comms.StageTickerData(exchangeName, assetType, result)
	if bot.config.Webserver.Enabled {
		relayWebsocketEvent(result, "ticker_update", assetType, exchangeName)
	}
	return nil
}

// ProcessOrderbookREST processes orderbook data, utilising a REST endpoint
func (m *Monitor) ProcessOrderbookREST(exch exchange.IBotExchange, p pair.CurrencyPair, assetType string, verbose bool) error {
	result, err := exch.UpdateOrderbook(p, assetType)
	printOrderbookSummary(result, p, assetType, exch.GetName(), err)
	if err != nil {
		return err
	}

	err = m.ValidateData(result)
	if err != nil {
		return err
	}

	bot.comms.StageOrderbookData(exch.GetName(), assetType, result)

	if bot.config.Webserver.Enabled {
		relayWebsocketEvent(result, "orderbook_update", assetType, exch.GetName())
	}
	return nil
}

// ProcessHistoryREST processes history data, utilising a REST endpoint
func (m *Monitor) ProcessHistoryREST(exch exchange.IBotExchange, p pair.CurrencyPair, assetType string, verbose bool) error {
	lastTime, tradeID, err := bot.db.GetExchangeTradeHistoryLast(exch.GetName(),
		p.Pair().String(),
		assetType)
	if err != nil {
		log.Fatal(err)
	}

	if time.Now().Truncate(5*time.Minute).Unix() < lastTime.Unix() {
		return errors.New("history up to date")
	}

	result, err := exch.GetExchangeHistory(p,
		assetType,
		lastTime,
		tradeID)
	if err != nil {
		return err
	}

	if len(result) < 1 {
		return errors.New("no history returned")
	}

	for i := range result {
		err := bot.db.InsertExchangeTradeHistoryData(result[i].TID,
			result[i].Exchange,
			p.Pair().String(),
			assetType,
			result[i].Type,
			result[i].Amount,
			result[i].Price,
			result[i].Timestamp)
		if err != nil {
			if err.Error() == "row already found" {
				continue
			}
			log.Fatal(err)
		}
	}
	return nil
}

// ValidateData validates incoming from either a REST endpoint or websocket feed
// by its type
func (m *Monitor) ValidateData(i interface{}) error {
	switch i.(type) {
	case ticker.Price:
		if i.(ticker.Price).Ask == 0 && i.(ticker.Price).Bid == 0 &&
			i.(ticker.Price).High == 0 && i.(ticker.Price).Last == 0 &&
			i.(ticker.Price).Low == 0 && i.(ticker.Price).PriceATH == 0 &&
			i.(ticker.Price).Volume == 0 {
			return errors.New("routines.go ValidateData() error - insufficient returned data")
		}
		if i.(ticker.Price).Volume == 0 {
			return errors.New("routines.go ValidateData() error - volume assignment error")
		}
		if i.(ticker.Price).Last == 0 {
			return errors.New("routines.go ValidateData() error - critcal value: Last Price assignment error")
		}

	case orderbook.Base:
		if len(i.(orderbook.Base).Asks) == 0 || len(i.(orderbook.Base).Bids) == 0 {
			return errors.New("routines.go ValidateData() error - insufficient returned data")
		}

	default:
		return errors.New("routines.go ValidateData() error - can't handle data type")
	}
	return nil
}

// WebsocketRoutine Initial routine management system for websocket
func WebsocketRoutine(verbose bool) {
	log.Println("Connecting exchange websocket services...")

	for i := range bot.exchanges {
		go func(i int) {
			if verbose {
				log.Printf("Establishing websocket connection for %s",
					bot.exchanges[i].GetName())
			}

			ws, err := bot.exchanges[i].GetWebsocket()
			if err != nil {
				return
			}

			// Data handler routine
			go WebsocketDataHandler(ws, verbose)

			err = ws.Connect()
			if err != nil {
				switch err.Error() {
				case exchange.WebsocketNotEnabled:
					// Store in memory if enabled in future
				default:
					log.Println(err)
				}
			}
		}(i)
	}
}

var shutdowner = make(chan struct{}, 1)
var wg sync.WaitGroup

// Websocketshutdown shuts down the exchange routines and then shuts down
// governing routines
func Websocketshutdown(ws *exchange.Websocket) error {
	err := ws.Shutdown() // shutdown routines on the exchange
	if err != nil {
		log.Fatalf("routines.go error - failed to shutodwn %s", err)
	}

	timer := time.NewTimer(5 * time.Second)
	c := make(chan struct{}, 1)

	go func(c chan struct{}) {
		close(shutdowner)
		wg.Wait()
		c <- struct{}{}
	}(c)

	select {
	case <-timer.C:
		return errors.New("routines.go error - failed to shutdown routines")

	case <-c:
		return nil
	}
}

// streamDiversion is a diversion switch from websocket to REST or other
// alternative feed
func streamDiversion(ws *exchange.Websocket, verbose bool) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-shutdowner:
			return

		case <-ws.Connected:
			if verbose {
				log.Printf("exchange %s websocket feed connected", ws.GetName())
			}

			update.SwitchRESTFunctionality(true, ws.GetName())

		case <-ws.Disconnected:
			if verbose {
				log.Printf("exchange %s websocket feed disconnected, switching to REST functionality",
					ws.GetName())
			}

			update.SwitchRESTFunctionality(false, ws.GetName())
		}
	}
}

// WebsocketDataHandler handles websocket data coming from a websocket feed
// associated with an exchange
func WebsocketDataHandler(ws *exchange.Websocket, verbose bool) {
	wg.Add(1)
	defer wg.Done()

	go streamDiversion(ws, verbose)

	for {
		select {
		case <-shutdowner:
			return

		case data := <-ws.DataHandler:
			switch data.(type) {
			case string:
				switch data.(string) {
				case exchange.WebsocketNotEnabled:
					if verbose {
						log.Printf("routines.go warning - exchange %s weboscket not enabled",
							ws.GetName())
					}

				default:
					log.Println(data.(string))
				}

			case error:
				switch {
				case common.StringContains(data.(error).Error(), "close 1006"):
					go WebsocketReconnect(ws, verbose)
					continue
				default:
					log.Fatalf("routines.go exchange %s websocket error - %s", ws.GetName(), data)
				}

			case exchange.TradeData:
				// Trade Data
				if verbose {
					log.Println("Websocket trades Updated:   ", data.(exchange.TradeData))
				}

			case exchange.TickerData:
				// Ticker data
				if verbose {
					log.Println("Websocket Ticker Updated:   ", data.(exchange.TickerData))
				}

			case exchange.KlineData:
				// Kline data
				if verbose {
					log.Println("Websocket Kline Updated:    ", data.(exchange.KlineData))
				}

			case exchange.WebsocketOrderbookUpdate:
				// Orderbook data
				if verbose {
					log.Println("Websocket Orderbook Updated:", data.(exchange.WebsocketOrderbookUpdate))
				}

			default:
				if verbose {
					log.Println("Websocket Unknown type:     ", data)
				}
			}
		}
	}
}

// WebsocketReconnect tries to reconnect to a websocket stream
func WebsocketReconnect(ws *exchange.Websocket, verbose bool) {
	if verbose {
		log.Printf("Websocket reconnection requested for %s", ws.GetName())
	}

	err := ws.Shutdown()
	if err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	defer wg.Done()

	ticker := time.NewTicker(3 * time.Second)
	for {
		select {
		case <-shutdowner:
			return

		case <-ticker.C:
			err = ws.Connect()
			if err == nil {
				return
			}
		}
	}
}
